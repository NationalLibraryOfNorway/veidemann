import {DataSource} from '@angular/cdk/collections';
import {DestroyRef} from '@angular/core';
import {BehaviorSubject, EMPTY, Observable, Subject, Subscription} from 'rxjs';
import {catchError, distinctUntilChanged, finalize, tap} from 'rxjs/operators';

export const LIST_PAGE_SIZE = 100;

export interface ListItem {
  id: string;
}

export interface ListRange {
  offset: number;
  pageSize: number;
}

export interface ListDataSourceOptions<Q, T extends ListItem> {
  query$: Observable<Q>;
  load: (query: Q, range: ListRange) => Observable<T>;
  destroyRef: DestroyRef;
  /** A positive capacity denotes a bounded, non-completing live query. */
  capacity?: (query: Q) => number;
  pageSize?: number;
}

export interface ListLoadCompleted<Q, T extends ListItem> {
  query: Q;
  rows: readonly T[];
  range: ListRange;
  received: number;
}

export interface ListReloadOptions {
  retainRows?: boolean;
}

export type ListComparator<T> = (left: T, right: T) => number;
export type ListPredicate<T> = (item: T) => boolean;

/**
 * Renderer-neutral state for streamed lists.
 *
 * The query owns filters/watch/server sort. This class alone owns ranges and
 * accumulated rows, so table, list and future card renderers behave alike.
 */
export class ListDataSource<T extends ListItem, Q = never> implements DataSource<T> {
  private readonly data = new BehaviorSubject<readonly T[]>([]);
  private readonly initialLoading = new BehaviorSubject(false);
  private readonly appendLoading = new BehaviorSubject(false);
  private readonly exhausted = new BehaviorSubject(false);
  private readonly appendFailed = new BehaviorSubject(false);
  private readonly reset = new Subject<void>();
  private readonly completed = new Subject<ListLoadCompleted<Q, T>>();
  private readonly destroy = new Subject<void>();

  private accumulated: readonly T[] = [];
  private latestQuery: Q;
  private hasQuery = false;
  private generation = 0;
  private offset = 0;
  private capacity = 0;
  private readonly pageSize: number;
  private comparator: ListComparator<T> | null = null;
  private predicate: ListPredicate<T> | null = null;
  private activeLoad = Subscription.EMPTY;
  private querySubscription = Subscription.EMPTY;
  private failedRange: ListRange | null = null;

  readonly rows$ = this.data.asObservable();
  readonly initialLoading$ = this.initialLoading.asObservable().pipe(distinctUntilChanged());
  readonly appendLoading$ = this.appendLoading.asObservable().pipe(distinctUntilChanged());
  readonly loading$: Observable<boolean>;
  readonly exhausted$ = this.exhausted.asObservable().pipe(distinctUntilChanged());
  readonly appendFailed$ = this.appendFailed.asObservable().pipe(distinctUntilChanged());
  readonly reset$ = this.reset.asObservable();
  readonly completed$ = this.completed.asObservable();

  private constructor(private readonly options: ListDataSourceOptions<Q, T>) {
    this.pageSize = options.pageSize ?? LIST_PAGE_SIZE;
    this.loading$ = new Observable<boolean>(subscriber => {
      const emit = () => subscriber.next(this.initialLoading.value || this.appendLoading.value);
      const subscriptions = [this.initialLoading.subscribe(emit), this.appendLoading.subscribe(emit)];
      return () => subscriptions.forEach(subscription => subscription.unsubscribe());
    }).pipe(distinctUntilChanged());
  }

  static fromQuery<Q, T extends ListItem>(options: ListDataSourceOptions<Q, T>): ListDataSource<T, Q> {
    const dataSource = new ListDataSource<T, Q>(options);
    dataSource.connectQuery();
    return dataSource;
  }

  get length(): number {
    return this.accumulated.length;
  }

  get snapshot(): readonly T[] {
    return this.data.value;
  }

  /** All rows accumulated from the server, before client-side filtering and sorting. */
  get loadedSnapshot(): readonly T[] {
    return this.accumulated;
  }

  connect(): Observable<readonly T[]> {
    return this.rows$;
  }

  disconnect(): void {
    return;
  }

  /** Changes only the derived view; accumulated server rows remain immutable. */
  setComparator(comparator: ListComparator<T> | null): void {
    this.comparator = comparator;
    this.publish();
  }

  /** Changes only the derived view; accumulated server rows remain immutable. */
  setPredicate(predicate: ListPredicate<T> | null): void {
    this.predicate = predicate;
    this.publish();
  }

  loadMore(): void {
    if (!this.hasQuery || this.capacity > 0 || this.initialLoading.value ||
      this.appendLoading.value || this.exhausted.value || this.appendFailed.value) {
      return;
    }
    this.request({offset: this.offset, pageSize: this.pageSize}, true);
  }

  retry(): void {
    if (!this.hasQuery || !this.failedRange || this.initialLoading.value || this.appendLoading.value) {
      return;
    }
    const range = this.failedRange;
    this.failedRange = null;
    this.appendFailed.next(false);
    this.request(range, range.offset > 0 || this.accumulated.length > 0);
  }

  reload(options: ListReloadOptions = {}): void {
    if (!this.hasQuery) {
      return;
    }
    ++this.generation;
    this.activeLoad.unsubscribe();
    this.offset = 0;
    this.failedRange = null;
    this.appendFailed.next(false);
    this.exhausted.next(false);
    if (!options.retainRows) {
      this.clear();
      this.reset.next();
    }
    this.request({offset: 0, pageSize: this.pageSize}, false);
  }

  /** Refreshes every currently loaded page and keeps the existing rows visible until replacement completes. */
  refreshLoaded(): void {
    if (!this.hasQuery || this.capacity > 0 || this.initialLoading.value || this.appendLoading.value) {
      return;
    }
    ++this.generation;
    this.activeLoad.unsubscribe();
    this.failedRange = null;
    this.appendFailed.next(false);
    this.requestReplacement({offset: 0, pageSize: Math.max(this.offset, this.pageSize)});
  }

  private connectQuery(): void {
    this.querySubscription = this.options.query$.subscribe(query => {
      this.latestQuery = query;
      this.hasQuery = true;
      this.capacity = this.options.capacity?.(query) ?? 0;
      this.reload();
    });

    this.options.destroyRef.onDestroy(() => {
      this.querySubscription.unsubscribe();
      this.activeLoad.unsubscribe();
      this.destroy.next();
      this.destroy.complete();
      this.initialLoading.complete();
      this.appendLoading.complete();
      this.exhausted.complete();
      this.appendFailed.complete();
      this.reset.complete();
      this.completed.complete();
      this.data.complete();
    });
  }

  private request(range: ListRange, append: boolean): void {
    const generation = this.generation;
    let received = 0;
    let completed = false;
    let failed = false;

    (append ? this.appendLoading : this.initialLoading).next(true);
    this.activeLoad = this.options.load(this.latestQuery, range).pipe(
      tap({
        next: item => {
          if (generation !== this.generation || !item) {
            return;
          }
          received++;
          this.upsert(item);
          // A watch stream may never complete; stop presenting it as an initial load
          // once the first live row is usable.
          if (this.capacity > 0) {
            this.initialLoading.next(false);
          }
        },
        complete: () => completed = true,
      }),
      catchError(() => {
        failed = true;
        return EMPTY;
      }),
      finalize(() => {
        if (generation !== this.generation) {
          return;
        }
        (append ? this.appendLoading : this.initialLoading).next(false);
        if (failed) {
          this.failedRange = range;
          this.appendFailed.next(true);
          return;
        }
        if (!completed) {
          return;
        }
        this.offset += received;
        this.exhausted.next(this.capacity > 0 || received < range.pageSize);
        this.completed.next({query: this.latestQuery, rows: this.snapshot, range, received});
      })
    ).subscribe();
  }

  private requestReplacement(range: ListRange): void {
    const generation = this.generation;
    const rows: T[] = [];
    let completed = false;

    this.activeLoad = this.options.load(this.latestQuery, range).pipe(
      tap({
        next: item => {
          if (generation !== this.generation || !item) {
            return;
          }
          const index = rows.findIndex(row => row.id === item.id);
          if (index >= 0) {
            rows[index] = item;
          } else {
            rows.push(item);
          }
        },
        complete: () => completed = true,
      }),
      catchError(() => EMPTY),
      finalize(() => {
        if (generation !== this.generation || !completed) {
          return;
        }
        this.accumulated = rows;
        this.offset = rows.length;
        this.exhausted.next(rows.length < range.pageSize);
        this.publish();
        this.completed.next({
          query: this.latestQuery,
          rows: this.snapshot,
          range,
          received: rows.length,
        });
      })
    ).subscribe();
  }

  private upsert(item: T): void {
    const index = this.accumulated.findIndex(row => row.id === item.id);
    if (index >= 0) {
      this.accumulated = this.accumulated.map((row, rowIndex) => rowIndex === index ? item : row);
    } else if (this.capacity > 0 && this.accumulated.length >= this.capacity) {
      this.accumulated = [item, ...this.accumulated.slice(0, this.capacity - 1)];
    } else {
      this.accumulated = [...this.accumulated, item];
    }
    this.publish();
  }

  private publish(): void {
    let view = this.predicate ? this.accumulated.filter(this.predicate) : [...this.accumulated];
    if (this.comparator) {
      view = view.sort(this.comparator);
    }
    this.data.next(view);
  }

  private clear(): void {
    this.accumulated = [];
    this.publish();
  }
}
