import {Observable, Observer} from 'rxjs';

export function fromServerStream<T>(
  streamFactory: (signal: AbortSignal) => AsyncIterable<T>,
): Observable<T> {
  return new Observable<T>((observer: Observer<T>) => {
    const abortController = new AbortController();

    void (async () => {
      try {
        for await (const message of streamFactory(abortController.signal)) {
          observer.next(message);
        }
        observer.complete();
      } catch (error) {
        if (!abortController.signal.aborted) {
          observer.error(error);
        }
      }
    })();

    return () => abortController.abort();
  });
}
