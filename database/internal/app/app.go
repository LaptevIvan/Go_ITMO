package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/project/library/config"
	"github.com/project/library/internal/entity"
	"github.com/project/library/internal/usecase/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/project/library/db"

	"runtime"

	gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	generated "github.com/project/library/generated/api/library"
	"github.com/project/library/internal/controller"
	"github.com/project/library/internal/usecase/library"
	"github.com/project/library/internal/usecase/repository"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	shutDownSeconds        = 3
	dialerTimeoutSeconds   = 30
	dialerKeepAliveSeconds = 180
	transportMaxIdleConns  = 100
	transportMaxConnsPerHost
	transportIdleConnTimeoutSeconds       = 90
	transportTLSHandshakeTimeoutSeconds   = 15
	transportExpectContinueTimeoutSeconds = 2
)

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dbPool, err := pgxpool.New(ctx, cfg.PG.URL)
	if err != nil {
		logger.Error("can not create pgxpool", zap.Error(err))
		return
	}
	defer dbPool.Close()
	db.SetupPostgres(dbPool, logger)

	var logRepo *zap.Logger
	if cfg.Log.LogDBRepo {
		logRepo = logger
	} else {
		logRepo = nil
	}
	repo := repository.New(logRepo, dbPool)
	outboxRepository := repository.NewOutbox(dbPool, cfg.Outbox.AttemptsRetry)

	var logTransactor *zap.Logger
	if cfg.Log.LogUseCase {
		logTransactor = logger
	} else {
		logTransactor = nil
	}
	transactor := repository.NewTransactor(logTransactor, dbPool)
	runOutbox(ctx, cfg, logger, outboxRepository, transactor)

	var logUseCase *zap.Logger
	if cfg.Log.LogUseCase {
		logUseCase = logger
	} else {
		logUseCase = nil
	}
	useCases := library.New(logUseCase, repo, repo, outboxRepository, transactor)

	var logController *zap.Logger
	if cfg.Log.LogController {
		logController = logger
	} else {
		logController = nil
	}
	ctrl := controller.New(logController, useCases, useCases)

	go runRest(ctx, cfg, logger)
	go runGrpc(cfg, logger, ctrl)

	<-ctx.Done()
	time.Sleep(time.Second * shutDownSeconds)
}

func runOutbox(
	ctx context.Context,
	cfg *config.Config,
	logger *zap.Logger,
	outboxRepository library.OutboxRepository,
	transactor repository.Transactor,
) {
	dialer := &net.Dialer{
		Timeout:   dialerTimeoutSeconds * time.Second,
		KeepAlive: dialerKeepAliveSeconds * time.Second,
	}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          transportMaxIdleConns,
		MaxConnsPerHost:       transportMaxConnsPerHost,
		IdleConnTimeout:       transportIdleConnTimeoutSeconds * time.Second,
		TLSHandshakeTimeout:   transportTLSHandshakeTimeoutSeconds * time.Second,
		ExpectContinueTimeout: transportExpectContinueTimeoutSeconds * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}

	client := new(http.Client)
	client.Transport = transport

	globalHandler := globalOutboxHandler(client, cfg.Outbox.AuthorSendURL, cfg.Outbox.BookSendURL)

	var logOutbox *zap.Logger
	if cfg.Log.LogOutboxWorker {
		logOutbox = logger
	} else {
		logOutbox = nil
	}
	outboxService := outbox.New(logOutbox, outboxRepository, globalHandler, cfg, transactor)

	outboxService.Start(
		ctx,
		cfg.Outbox.Workers,
		cfg.Outbox.BatchSize,
		cfg.Outbox.WaitTimeMS,
		cfg.Outbox.InProgressTTLMS,
	)
}

func globalOutboxHandler(
	client *http.Client,
	authorURL,
	bookURL string,
) outbox.GlobalHandler {
	return func(kind repository.OutboxKind) (outbox.KindHandler, error) {
		switch kind {
		case repository.OutboxKindAuthor:
			return authorOutboxHandler(client, authorURL), nil
		case repository.OutboxKindBook:
			return bookOutboxHandler(client, bookURL), nil
		default:
			return nil, fmt.Errorf("unsupported outbox kind: %d", kind)
		}
	}
}

const contentType = "application/json"

var errFailRequest = errors.New("Not 2xx response")

const statusOk = 2

func authorOutboxHandler(client *http.Client, url string) outbox.KindHandler {
	return func(_ context.Context, data []byte) error {
		author := entity.Author{}
		err := json.Unmarshal(data, &author)

		if err != nil {
			return fmt.Errorf("can not deserialize data in book outbox handler: %w", err)
		}

		response, err := client.Post(url, contentType, strings.NewReader(author.ID))
		if err != nil {
			return fmt.Errorf("can not make post request to given url: %w", err)
		}

		defer response.Body.Close()

		if response.StatusCode/100 != statusOk {
			return errFailRequest
		}

		return nil
	}
}

func bookOutboxHandler(client *http.Client, url string) outbox.KindHandler {
	return func(_ context.Context, data []byte) error {
		book := entity.Book{}
		err := json.Unmarshal(data, &book)

		if err != nil {
			return fmt.Errorf("can not deserialize data in book outbox handler: %w", err)
		}

		response, err := client.Post(url, contentType, strings.NewReader(book.ID))
		if err != nil {
			return fmt.Errorf("can not make post request to given url: %w", err)
		}

		defer response.Body.Close()

		if response.StatusCode/100 != statusOk {
			return errFailRequest
		}

		return nil
	}
}

func runRest(ctx context.Context, cfg *config.Config, logger *zap.Logger) {
	mux := gateway.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	address := "localhost:" + cfg.GRPC.Port
	err := generated.RegisterLibraryHandlerFromEndpoint(ctx, mux, address, opts)

	if err != nil {
		logger.Error("can not register grpc gateway", zap.Error(err))
		os.Exit(-1)
	}

	gatewayPort := ":" + cfg.GRPC.GatewayPort
	logger.Info("gateway listening at port", zap.String("port", gatewayPort))

	if err = http.ListenAndServe(gatewayPort, mux); err != nil {
		logger.Error("gateway listen error", zap.Error(err))
	}
}

func runGrpc(cfg *config.Config, logger *zap.Logger, libraryService generated.LibraryServer) {
	port := ":" + cfg.GRPC.Port
	lis, err := net.Listen("tcp", port)

	if err != nil {
		logger.Error("can not open tcp socket", zap.Error(err))
		os.Exit(-1)
	}

	s := grpc.NewServer()
	reflection.Register(s)

	generated.RegisterLibraryServer(s, libraryService)

	logger.Info("grpc server listening at port", zap.String("port", port))

	if err = s.Serve(lis); err != nil {
		logger.Error("grpc server listen error", zap.Error(err))
	}
}
