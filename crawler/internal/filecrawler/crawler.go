package crawler

import (
	"context"
	"crawler/internal/fs"
	"crawler/internal/workerpool"
	"encoding/json"
	"fmt"
)

// Configuration holds the configuration for the crawler, specifying the number of workers for
// file searching, processing, and accumulating tasks. The values for SearchWorkers, FileWorkers,
// and AccumulatorWorkers are critical to efficient performance and must be defined in
// every configuration.
type Configuration struct {
	SearchWorkers      int // Number of workers responsible for searching files.
	FileWorkers        int // Number of workers for processing individual files.
	AccumulatorWorkers int // Number of workers for accumulating results.
}

// Combiner is a function type that defines how to combine two values of type R into a single
// result. Combiner is not required to be thread-safe
//
// Combiner can either:
//   - Modify one of its input arguments to include the result of the other and return it,
//     or
//   - Create a new combined result based on the inputs and return it.
//
// It is assumed that type R has a neutral element (forming a monoid)
type Combiner[R any] func(current R, accum R) R

// Crawler represents a concurrent crawler implementing a map-reduce model with multiple workers
// to manage file processing, transformation, and accumulation tasks. The crawler is designed to
// handle large sets of files efficiently, assuming that all files can fit into memory
// simultaneously.
type Crawler[T, R any] interface {
	// Collect performs the full crawling operation, coordinating with the file system
	// and worker pool to process files and accumulate results. The result type R is assumed
	// to be a monoid, meaning there exists a neutral element for combination, and that
	// R supports an associative combiner operation.
	// The result of this collection process, after all reductions, is returned as type R.
	//
	// Important requirements:
	// 1. Number of workers in the Configuration is mandatory for managing workload efficiently.
	// 2. FileSystem and Accumulator must be thread-safe.
	// 3. Combiner does not need to be thread-safe.
	// 4. If an accumulator or combiner function modifies one of its arguments,
	//    it should return that modified value rather than creating a new one,
	//    or alternatively, it can create and return a new combined result.
	// 5. Context cancellation is respected across workers.
	// 6. Type T is derived by json-deserializing the file contents, and any issues in deserialization
	//    must be handled within the worker.
	// 7. The combiner function will wait for all workers to complete, ensuring no goroutine leaks
	//    occur during the process.
	Collect(
		ctx context.Context,
		fileSystem fs.FileSystem,
		root string,
		conf Configuration,
		accumulator workerpool.Accumulator[T, R],
		combiner Combiner[R],
	) (R, error)
}

type crawlerImpl[T, R any] struct{}

func New[T, R any]() *crawlerImpl[T, R] {
	return &crawlerImpl[T, R]{}
}

// The function which uses to catch panic. It writes error about panic to inputted chan
var catch = func(output chan error) {
	if x := recover(); x != nil {
		switch tp := x.(type) {
		case error: // if value in caught panic is error
			output <- tp // writes its
		default:
			output <- fmt.Errorf("Stopped due to panic: %#v", tp) // creates its own panic error
		}
	}
}

// The function searches from root directory all files and returns output chan of paths to these files.
// Besides tools for working it accepts chan of error. It will write caught error or error about panic to this chan
// so that called function can determine whether there was an error
func (c *crawlerImpl[T, R]) search(ctx context.Context, workers int, root string, fileSystem fs.FileSystem, err chan error) <-chan string {
	files := make(chan string) // output chan of paths to found files
	go func() {
		defer close(files)                                               // asynchronous closes the channel
		poolSearch := workerpool.New[string, string]()                   // creates workerpool
		poolSearch.List(ctx, workers, root, func(node string) []string { // uses its method List
			defer catch(err)                       // catches panic and writes about it to inputted chan err
			entries, e := fileSystem.ReadDir(node) // gets []os.DirEntry by fileSystems
			if e != nil {                          // if there was an error
				err <- e // stops the working
				return nil
			}
			ans := make([]string, 0) // creates slice of child elements as Searcher function
			for _, entry := range entries {
				path := fileSystem.Join(node, entry.Name()) // creates path to considered os.DirEntry
				if entry.IsDir() {                          // if it is directory
					ans = append(ans, path) // appends it to child elements to considered its to next layer
				} else {
					select {
					case <-ctx.Done(): // stops if context is closed
					case files <- path: // tries to write found path to files
					}
				}
			}
			return ans // returns child elements
		})
	}()
	return files // returns output chan
}

// The function deserializes found files to type T and returns chan of processed values of type T.
// Besides tools for working it accepts chan of error. It will write caught error or error about panic to this chan
// so that called function can determine whether there was an error
func (c *crawlerImpl[T, R]) makeDeserialization(ctx context.Context, workers int, inp <-chan string, fileSystem fs.FileSystem, err chan error) <-chan T {
	poolTransform := workerpool.New[string, T]()                                  // creates workerpool
	jsons := poolTransform.Transform(ctx, workers, inp, func(filePath string) T { // uses its method Transform
		defer catch(err)                     // catches panic and writes about it to inputted chan err
		file, e := fileSystem.Open(filePath) // opens inputted file to deserialization
		var t T
		defer func() { // delayed file closure
			if file != nil {
				if e = file.Close(); e != nil { // Tries to close file and  if it fails
					println("Error ", e.Error(), " closing the file by path: ", filePath) // logs it to stderr
				}
			}
		}()
		if e != nil { // if there was an error opening the file
			err <- e // writes error to inputted chan
			return t // returns null value of type T
		}
		e = json.NewDecoder(file).Decode(&t) // does deserialization by json decoder
		if e != nil {                        // if there was an error opening the file
			err <- e // writes error to inputted chan
		}
		return t // returns processed value of type T
	})
	return jsons // returns output chan
}

// The function creates worker that combines accumulated values of type R from different workers to one result value.
// Functions returns chan and later will write result value to its
func (c *crawlerImpl[T, R]) combineValuesR(ctx context.Context, workers int, inp <-chan T, accumulator workerpool.Accumulator[T, R], combiner Combiner[R]) chan R {
	res := make(chan R) // output chan
	go func() {
		defer close(res)                                                    // asynchronous closes the channel
		accumPool := workerpool.New[T, R]()                                 // creates workerpool
		accumValues := accumPool.Accumulate(ctx, workers, inp, accumulator) // uses its method Accumulate
		var accum R                                                         // default value: the neutral element of type R
		for r := range accumValues {                                        // while chan accumValues isn't closed
			accum = combiner(r, accum) // combines values from its
		}
		select {
		case <-ctx.Done(): // context is closed
			return // stop working
		case res <- accum: // tries to write result value to res
		}
	}()
	return res // returns output chan
}

func (c *crawlerImpl[T, R]) Collect(
	ctx context.Context,
	fileSystem fs.FileSystem,
	root string,
	conf Configuration,
	accumulator workerpool.Accumulator[T, R],
	combiner Combiner[R],
) (R, error) {
	ctxErr, cancel := context.WithCancelCause(ctx) // creates from ctx new context with cancel function that accepts error - reason of canceling

	// creates a new context from ctxErr without the undo function so that workers in the pipeline
	// do not stop their work earlier than workers at the beginning of the pipeline. They will stop
	// due to the closure of the channel transferred to them, which will be closed by workers working
	// with the first channel of the conveyor. This allows to avoid leakage of goroutines
	ctxForPipeline := context.WithoutCancel(ctxErr)

	err := make(chan error)                                                                        // chan so that workers in the pipeline can write the error that occurred to its
	defer close(err)                                                                               // closes the chan err after return
	files := c.search(ctxErr, conf.SearchWorkers, root, fileSystem, err)                           // chan of paths to files in directory root (and subdirectories)
	jsons := c.makeDeserialization(ctxForPipeline, conf.FileWorkers, files, fileSystem, err)       // channel with json deserialized file values
	res := c.combineValuesR(ctxForPipeline, conf.AccumulatorWorkers, jsons, accumulator, combiner) // result chan with one result value

	for {
		select {
		case e := <-err: // if there was an error in something worker
			cancel(e) // calls cancel function with this error
		case val := <-res: // if result value is calculated
			return val, context.Cause(ctxErr) // returns this value and (perhaps) happened error
		}
	}
}
