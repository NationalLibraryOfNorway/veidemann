import {BehaviorSubject, Observable} from 'rxjs';
import {distinctUntilChanged, finalize, map, scan} from 'rxjs/operators';

export interface Loader {
  loading$: Observable<boolean>;
}

export abstract class LoadingService implements Loader {
  private readonly loading = new BehaviorSubject<boolean>(false);

  readonly loading$: Observable<boolean> = this.loading.pipe(
    map(isLoading => isLoading ? 1 : -1),
    scan((acc, curr) => acc + curr, 0),
    map(sem => sem > 0),
    distinctUntilChanged()
  );

  protected load<T>(observable: Observable<T>): Observable<T> {
    this.loading.next(true);
    return observable.pipe(finalize(() => this.loading.next(false)));
  }
}
