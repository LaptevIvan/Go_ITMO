package workerpool

import (
	"context"
	"sync"
)

// Accumulator is a function type used to aggregate values of type T into a result of type R.
// It must be thread-safe, as multiple goroutines will access the accumulator function concurrently.
// Each worker will produce intermediate results, which are combined with an initial or
// accumulated value.
type Accumulator[T, R any] func(current T, accum R) R

// Transformer is a function type used to transform an element of type T to another type R.
// The function is invoked concurrently by multiple workers, and therefore must be thread-safe
// to ensure data integrity when accessed across multiple goroutines.
// Each worker independently applies the transformer to its own subset of data, and although
// no shared state is expected, the transformer must handle any internal state in a thread-safe
// manner if present.
type Transformer[T, R any] func(current T) R

// Searcher is a function type for exploring data in a hierarchical manner.
// Each call to Searcher takes a parent element of type T and returns a slice of T representing
// its child elements. Since multiple goroutines may call Searcher concurrently, it must be
// thread-safe to ensure consistent results during recursive  exploration.
//
// Important considerations:
//  1. Searcher should be designed to avoid race conditions, particularly if it captures external
//     variables in closures.
//  2. The calling function must handle any state or values in closures, ensuring that
//     captured variables remain consistent throughout recursive or hierarchical search paths.
type Searcher[T any] func(parent T) []T

// Pool is the primary interface for managing worker pools, with support for three main
// operations: Transform, Accumulate, and List. Each operation takes an input channel, applies
// a transformation, accumulation, or list expansion, and returns the respective output.
type Pool[T, R any] interface {
	// Transform applies a transformer function to each item received from the input channel,
	// with results sent to the output channel. Transform operates concurrently, utilizing the
	// specified number of workers. The number of workers must be explicitly defined in the
	// configuration for this function to handle expected workloads effectively.
	// Since multiple workers may call the transformer function concurrently, it must be
	// thread-safe to prevent race conditions or unexpected results when handling shared or
	// internal state. Each worker independently applies the transformer function to its own
	// data subset.
	Transform(ctx context.Context, workers int, input <-chan T, transformer Transformer[T, R]) <-chan R

	// Accumulate applies an accumulator function to the items received from the input channel,
	// with results accumulated and sent to the output channel. The accumulator function must
	// be thread-safe, as multiple workers concurrently update the accumulated result.
	// The output channel will contain intermediate accumulated results as R
	Accumulate(ctx context.Context, workers int, input <-chan T, accumulator Accumulator[T, R]) <-chan R

	// List expands elements based on a searcher function, starting
	// from the given element. The searcher function finds child elements for each parent,
	// allowing exploration in a tree-like structure.
	// The number of workers should be configured based on the workload, ensuring each worker
	// independently processes assigned elements.
	List(ctx context.Context, workers int, start T, searcher Searcher[T])
}

type poolImpl[T, R any] struct{}

func New[T, R any]() *poolImpl[T, R] {
	return &poolImpl[T, R]{}
}

func (p *poolImpl[T, R]) Accumulate(
	ctx context.Context,
	workers int,
	input <-chan T,
	accumulator Accumulator[T, R],
) <-chan R {
	ans := make(chan R)    // output chan with intermediate results from workers
	wg := sync.WaitGroup{} // sync.WaitGroup for asynchronous closing the channel

	for range workers { // cycle for making workers
		wg.Add(1) // increments score in wg
		go func() {
			defer wg.Done() // decrements score in wg
			var accum R     // default value: the neutral element of type R
			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-input:
					if !ok { // if input chan is closed
						select {
						case <-ctx.Done(): // if context is closed
							return // stops the working
						case ans <- accum: // worker tries to write its intermediate result
							return
						}
					}
					select {
					case <-ctx.Done(): // if context is closed
						return // stops the working
					default:
						accum = accumulator(v, accum) // accumulates intermediate result
					}
				}
			}
		}()
	}
	go func() {
		defer close(ans) // asynchronous closes the channel
		wg.Wait()
	}()

	return ans
}

// The function generator returns chan that returns values from inputted slice 'values'
func (p *poolImpl[T, R]) generator(ctx context.Context, values []T) <-chan T {
	ans := make(chan T) // output chan

	go func() { // creates worker that asynchronous writes values from inputted slice 'values' to chan
		defer close(ans) // asynchronous closes the channel
		for _, v := range values {
			select {
			case <-ctx.Done():
				return // stops the working if context is closed
			case ans <- v: // write v from values to ans
			}
		}
	}()

	return ans
}

// The function listWorker creates worker that will return chan of found child elements
func (p *poolImpl[T, R]) listWorker(ctx context.Context, inp <-chan T, searcher Searcher[T]) <-chan T {
	nextNodes := make([]T, 0) // slice of found child elements
	for {
		select {
		case <-ctx.Done(): // is context is closed
			return nil // returns nil and stops the work
		case v, ok := <-inp:
			if !ok {
				if len(nextNodes) > 0 { // if there are some found child elements
					return p.generator(ctx, nextNodes) // returns generated slice that returns element from nextNode
				}
				return nil // returns nil
			}
			found := searcher(v)                    // searches child elements
			nextNodes = append(nextNodes, found...) // added them to nextNodes
		}
	}
}

// The function fanIn unions the slice of channels to one chan by pattern Fan-in
func (p *poolImpl[T, R]) fanIn(ctx context.Context, channels []<-chan T) <-chan T {
	ans := make(chan T)    // output chan that returns values from inputted channels
	wg := sync.WaitGroup{} // sync.WaitGroup for asynchronous closing the channel

	wg.Add(1) // increments score in wg
	go func() {
		defer wg.Done()               // decrements score in wg
		for _, ch := range channels { // every chan in channels reads one worker
			wg.Add(1) // increments score in wg
			go func() {
				defer wg.Done() // decrements score in wg

				select {
				case <-ctx.Done():
					return
				case v, ok := <-ch:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case ans <- v: // writes value to ans
					}
				}
			}()
		}
	}()

	go func() {
		defer close(ans) // asynchronous closes the channel
		wg.Wait()
	}()

	return ans
}

func (p *poolImpl[T, R]) List(ctx context.Context, workers int, start T, searcher Searcher[T]) {
	select {
	case <-ctx.Done(): // if context is already closed
		return // stop the working
	default:
		found := searcher(start)       // finds child elements of start
		inp := p.generator(ctx, found) // creates chan of this elements

		wg := sync.WaitGroup{} // sync.WaitGroup for wait ending of working with tree's layer
		rw := sync.RWMutex{}   // RWMutex to thread safe writing to slice channels
	loop:
		for {
			channels := make([]<-chan T, 0) // slice of channels, each of which will transmit the elements that 1 worker found
			for i := 0; i < workers; i++ {  // generates workers which will find child of current found elements
				wg.Add(1) // increments score in wg
				go func() {
					defer wg.Done() // decrements score in wg
					ch := p.listWorker(ctx, inp, searcher)
					if ch != nil { // if worker found some child elements
						rw.Lock()                       // makes a lock to perform thread-safe writing
						channels = append(channels, ch) // its chan added to channels
						rw.Unlock()                     // it is unlocked so that other goroutines can record
					}
				}()
			}
			wg.Wait() // waits until all workers return their channels
			select {
			case <-ctx.Done(): // if context is closed
				return // stops the working
			default:
				if len(channels) > 0 { // if there are some channels
					inp = p.fanIn(ctx, channels) // unions channels to one chan by pattern Fan-in
					continue                     // searches for child elements of the next layer
				}
				break loop // stops the working
			}
		}
	}
}

func (p *poolImpl[T, R]) Transform(
	ctx context.Context,
	workers int,
	input <-chan T,
	transformer Transformer[T, R],
) <-chan R {
	ans := make(chan R)    // output chan of transformed values of type R
	wg := sync.WaitGroup{} // sync.WaitGroup for asynchronous closing the channel

	for range workers { // does workers which will do transform
		wg.Add(1) // increments score in wg
		go func() {
			defer wg.Done() // decrements score in wg
			for {           // actions of every worker
				select {
				case <-ctx.Done(): // if context is closed
					return // worker stops
				case v, ok := <-input:
					if !ok {
						return
					}

					select {
					case <-ctx.Done(): // if context is closed
						return // worker stops
					case ans <- transformer(v): // writes to output chan
					}
				}
			}
		}()
	}
	go func() {
		defer close(ans) // asynchronous closes the channel
		wg.Wait()
	}()

	return ans // return output chan
}
