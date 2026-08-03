import {DataSource} from '@angular/cdk/collections';
import {BehaviorSubject, EMPTY, merge, Observable, Subject} from 'rxjs';
import {DestroyRef} from '@angular/core';
import {catchError, distinctUntilChanged, filter, finalize, ignoreElements, map, switchMap, takeUntil, tap} from 'rxjs/operators';

export interface ListItem {
  id: string;
}

export interface ListDataSourceOptions<Q, T extends ListItem> {
  query$: Observable<Q>;
  load: (query: Q) => Observable<T>;
  destroyRef: DestroyRef;
  capacity?: (query: Q) => number;
}

export interface ListLoadCompleted<Q, T extends ListItem> {
  query: Q;
  rows: readonly T[];
}

export interface ListReloadOptions {
  retainRows?: boolean;
}

interface LoadRequest<Q> {
  query: Q;
  retainRows: boolean;
}

interface ActiveLoad<Q> extends LoadRequest<Q> {
  generation: number;
}

export class ListDataSource<T extends ListItem, Q = never> implements DataSource<T> {
  private readonly data: BehaviorSubject<T[]>;
  private readonly loading: BehaviorSubject<boolean>;
  private readonly reset: Subject<void>;
  private readonly completed: Subject<ListLoadCompleted<Q, T>>;
  private readonly reloadRequest: Subject<ListReloadOptions>;
  private readonly destroy: Subject<void>;

  private latestQuery: Q;
  private hasQuery = false;
  private generation = 0;
  private capacity = 0;

  readonly rows$: Observable<readonly T[]>;
  readonly loading$: Observable<boolean>;
  readonly reset$: Observable<void>;
  readonly completed$: Observable<ListLoadCompleted<Q, T>>;

  private constructor() {
    this.data = new BehaviorSubject([]);
    this.loading = new BehaviorSubject(false);
    this.reset = new Subject();
    this.completed = new Subject();
    this.reloadRequest = new Subject();
    this.destroy = new Subject();
    this.rows$ = this.data.asObservable();
    this.loading$ = this.loading.asObservable().pipe(distinctUntilChanged());
    this.reset$ = this.reset.asObservable();
    this.completed$ = this.completed.asObservable();
  }

  static fromQuery<Q, T extends ListItem>(options: ListDataSourceOptions<Q, T>): ListDataSource<T, Q> {
    const dataSource = new ListDataSource<T, Q>();
    dataSource.connectQuery(options);
    return dataSource;
  }

  get length(): number {
    return this.data.value.length;
  }

  get snapshot(): readonly T[] {
    return this.data.value;
  }

  connect(): Observable<readonly T[]> {
    return this.rows$;
  }

  disconnect(): void {
    return;
  }

  reload(options: ListReloadOptions = {}): void {
    if (this.hasQuery) {
      this.reloadRequest.next(options);
    }
  }

  /**
   * Add item to internal data store
   *
   * If item with same id already exists it gets replaced.
   */
  private add(item: T) {
    if (!item) {
      return;
    }
    const found = this.data.value.find(c => c.id === item.id);
    if (found) {
      this.replace(item);
    } else {
      if (this.capacity && this.data.value.length >= this.capacity) {
        this.data.next([item, ...this.data.value.slice(0, -1)]);
      } else {
        this.data.next(this.data.value.concat(item));
      }
    }
  }

  /**
   * Replace an item in the internal store
   */
  private replace(item: T) {
    const index = this.data.value.findIndex(c => c.id === item.id);
    if (index !== -1) {
      this.data.next(this.data.value.map((value, valueIndex) => valueIndex === index ? item : value));
    }
  }

  private clear() {
    this.data.next([]);
  }

  private connectQuery(options: ListDataSourceOptions<Q, T>): void {
    const queryRequest$ = options.query$.pipe(
      tap(query => {
        this.latestQuery = query;
        this.hasQuery = true;
      }),
      map(query => ({query, retainRows: false}))
    );

    const reloadRequest$ = this.reloadRequest.pipe(
      filter(() => this.hasQuery),
      map(reloadOptions => ({query: this.latestQuery, retainRows: reloadOptions.retainRows === true}))
    );

    merge(queryRequest$, reloadRequest$).pipe(
      map(request => this.startLoad(request, options.capacity)),
      switchMap(request => this.load(request, options.load)),
      takeUntil(this.destroy)
    ).subscribe();

    options.destroyRef.onDestroy(() => {
      this.destroy.next();
      this.destroy.complete();
      this.loading.complete();
      this.reset.complete();
      this.completed.complete();
      this.reloadRequest.complete();
      this.data.complete();
    });
  }

  private startLoad(request: LoadRequest<Q>, capacity?: (query: Q) => number): ActiveLoad<Q> {
    const generation = ++this.generation;
    this.capacity = capacity ? capacity(request.query) : 0;
    this.loading.next(true);

    if (!request.retainRows) {
      this.clear();
      this.reset.next();
    }

    return {...request, generation};
  }

  private load(request: ActiveLoad<Q>, loader: (query: Q) => Observable<T>): Observable<never> {
    let completed = false;

    return loader(request.query).pipe(
      tap({
        next: item => this.add(item),
        complete: () => completed = true,
      }),
      catchError(() => EMPTY),
      finalize(() => {
        if (request.generation === this.generation) {
          this.loading.next(false);
          if (completed) {
            this.completed.next({query: request.query, rows: this.snapshot});
          }
        }
      }),
      ignoreElements()
    );
  }
}
