package fact

import (
	"errors"
	"fmt"
	"io"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

var (
	// ErrFactorizationCancelled is returned when the factorization process is cancelled via the done channel.
	ErrFactorizationCancelled = errors.New("cancelled")

	// ErrWriterInteraction is returned if an error occurs while interacting with the writer
	// triggering early termination.
	ErrWriterInteraction = errors.New("writer interaction")
)

// Config defines the configuration for factorization and write workers.
type Config struct {
	FactorizationWorkers int
	WriteWorkers         int
}

// Factorization interface represents a concurrent prime factorization task with configurable workers.
// Thread safety and error handling are implemented as follows:
// - The provided writer must be thread-safe to handle concurrent writes from multiple workers.
// - Output uses '\n' for newlines.
// - Factorization has a time complexity of O(sqrt(n)) per number.
// - If an error occurs while writing to the writer, early termination is triggered across all workers.
type Factorization interface {
	// Do performs factorization on a list of integers, writing the results to an io.Writer.
	// - done: a channel to signal early termination.
	// - numbers: the list of integers to factorize.
	// - writer: the io.Writer where factorization results are output.
	// - config: optional worker configuration.
	// Returns an error if the process is cancelled or if a writer error occurs.
	Do(done <-chan struct{}, numbers []int, writer io.Writer, config ...Config) error
}

// factorizationImpl provides an implementation for the Factorization interface.
type factorizationImpl struct{}

func New() *factorizationImpl {
	return &factorizationImpl{}
}

// The function does factorization of number n
func (f *factorizationImpl) factNum(n int) string {
	divisors := make([]string, 0, 1)
	curN := n
	if curN < 0 {
		divisors = append(divisors, "-1")
		if curN == math.MinInt { // checks that curN is a math.MinInt, to avoid overflow when replacing n with -n
			divisors = append(divisors, "2")
			curN /= 2 // reduces curN by dividing on 2
		}
		curN *= -1 // knowing that curN exactly isn't math.MinInt changes n to -n
	}
	supDiv := int(math.Sqrt(math.Abs(float64(n)))) // calculates the sqrt of to n to obtain the maximum value of the possible divisor
	i := 2
	for {
		if i > supDiv { // if i more then supDiv factorization is completed
			divisors = append(divisors, strconv.Itoa(curN))
			break
		}
		if (curN % i) == 0 { // The divisor of the number is found
			curN /= i
			divisors = append(divisors, strconv.Itoa(i))
			if i > curN { // if i more, then curN, factorization is completed
				break
			}
			continue
		}
		i++ // since i is not a divisor, it increases
	}
	return fmt.Sprintf("%s = %s\n", strconv.Itoa(n), strings.Join(divisors, " * "))
}

// The function creates FactorizationWorkers in the number of count Workers, which apply factNum to the number from the input chain and pass the result to the returned channel.
// Also, the function returns *sync.WaitGroup for caller can define ending of works convertNumToFact (and avoid goroutines leak).
func (f *factorizationImpl) convertNumToFact(countWorkers int, done <-chan struct{}, input <-chan int) (*sync.WaitGroup, <-chan string) {
	result := make(chan string)
	wg := new(sync.WaitGroup)
	for range countWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done: // if done is available for reading
					return // makes FactorizationCancelled
				case v, ok := <-input:
					if !ok { // if input chan is closed for reading
						return // makes end of work
					}
					select {
					case <-done: // checks for the last time that done is not open for reading
						return
					case result <- f.factNum(v): // applies factorization for number v
					}
				}
			}
		}()
	}

	go func() { // asynchronous channel closure
		defer close(result)
		wg.Wait()
	}()

	return wg, result
}

// The function creates Write Workers in the number of county Workers which write lines from the input channel to the writer.
// Returns *sync.WaitGroup for caller can define ending of works writeFact (and avoid goroutines leak)
// and <-chan error for caller can define whether there was an error in working.
func (f *factorizationImpl) writeFact(countWorkers int, done <-chan struct{}, input <-chan string, writer io.Writer) (*sync.WaitGroup, <-chan error) {
	wg := sync.WaitGroup{}
	once := sync.Once{}
	err := make(chan error, 1)
	doneCauseErrWriter := make(chan struct{})
	for range countWorkers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select { // checks if wasn't any error
				case <-done:
					return
				case <-doneCauseErrWriter:
					return
				case divisors, ok := <-input:
					if !ok {
						return
					}

					select { // checks for the last time that wasn't any error
					case <-done:
						return
					case <-doneCauseErrWriter:
						return
					default:
						_, errWriter := writer.Write([]byte(divisors))
						if errWriter != nil { // if was an error in working writer
							once.Do(func() {
								err <- fmt.Errorf("%w caused by %w", ErrWriterInteraction, errWriter)
								close(err)
								close(doneCauseErrWriter)
							})
						}
					}
				}
			}
		}()
	}

	return &wg, err
}

// Function checks that config was inputted and validates it or makes itself Config
func (f *factorizationImpl) makeConfig(config ...Config) (*Config, error) {
	var conf Config
	if len(config) > 0 { // if Config inputted
		conf = config[0]                                                // saves it
		if (conf.FactorizationWorkers < 1) || (conf.WriteWorkers < 1) { // validates the inputted configuration for the correctness of the values
			return nil, errors.New("incorrect value for config")
		}
	} else { // else makes its Config
		n := runtime.GOMAXPROCS(0) // As count workers for factorization and writings, picks current count of logical processors
		conf = Config{n, n}
	}
	return &conf, nil
}

func (f *factorizationImpl) Do(
	done <-chan struct{},
	numbers []int,
	writer io.Writer,
	config ...Config,
) error {
	conf, errConf := f.makeConfig(config...)
	if errConf != nil {
		return errConf
	}
	numCh := make(chan int, 1) // makes chan for write numbers from slice numbers. Chan is made with buffer size 1
	// so that in the future select can to write numbers to the channel without blocking
	wgFact, factNums := f.convertNumToFact(conf.FactorizationWorkers, done, numCh) // calls convertNumToFact so that there are workers which will read from numCh
	defer wgFact.Wait()                                                            // since done can become available for reading before calling writeFact, a call is made to wg.Wait() in defer here
	once := sync.Once{}
	select {
	case <-done:
		return ErrFactorizationCancelled
	default:
		// calls writeFact so that there are workers which will write factorizations from factNums to writer
		wgOut, err := f.writeFact(conf.WriteWorkers, done, factNums, writer)
		waitEnd := func() { // Function which waits end of work writeFact to avoid goroutines leak.
			once.Do(func() { // It works in defer if happens error in cycle is below, else after this cycle
				close(numCh)
				wgOut.Wait()
			})
		}
		defer waitEnd() // makes Defer to wait ending of the goroutines work in case of an error
		for i := 0; i < len(numbers); {
			select {
			case <-done: // checks that wasn't any error or cancel of factorization
				return ErrFactorizationCancelled
			case e := <-err:
				return e
			case numCh <- numbers[i]: // If it is possible to write to numCh
				i++
			default: // waits changes
			}
		}
		waitEnd() // waits for the end of the work

		select { // checks for the last time that wasn't any error
		case <-done:
			return ErrFactorizationCancelled
		case e := <-err:
			return e
		default:
			return nil
		}
	}
}
