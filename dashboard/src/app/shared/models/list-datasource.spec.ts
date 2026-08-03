import {DestroyRef} from '@angular/core';
import {BehaviorSubject, Observable, Subject} from 'rxjs';

import {ListDataSource, ListItem, ListRange} from './list-datasource';

interface TestItem extends ListItem {
  value: string;
}

class TestDestroyRef extends DestroyRef {
  private callbacks = new Set<() => void>();
  private isDestroyed = false;

  override get destroyed(): boolean {
    return this.isDestroyed;
  }

  override onDestroy(callback: () => void): () => void {
    this.callbacks.add(callback);
    return () => this.callbacks.delete(callback);
  }

  destroy(): void {
    this.isDestroyed = true;
    for (const callback of this.callbacks) {
      callback();
    }
    this.callbacks.clear();
  }
}

describe('ListDataSource', () => {
  it('requests the initial 100-row range and advances by raw server rows', () => {
    const query = new BehaviorSubject('query');
    const streams: Subject<TestItem>[] = [];
    const ranges: ListRange[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: (_query, range) => {
        ranges.push(range);
        const stream = new Subject<TestItem>();
        streams.push(stream);
        return stream;
      },
      destroyRef,
    });

    expect(ranges).toEqual([{offset: 0, pageSize: 100}]);
    for (let index = 0; index < 100; index++) {
      streams[0].next({id: String(index), value: String(index)});
    }
    streams[0].complete();
    dataSource.loadMore();

    expect(ranges.at(-1)).toEqual({offset: 100, pageSize: 100});
    streams[1].next({id: '99', value: 'replaced'});
    streams[1].next({id: '100', value: 'new'});
    streams[1].complete();

    expect(dataSource.length).toBe(101);
    expect(dataSource.snapshot.find(row => row.id === '99')?.value).toBe('replaced');
    dataSource.loadMore();
    expect(ranges).toHaveLength(2);
    destroyRef.destroy();
  });

  it('guards concurrent appends and retries a failed append without dropping rows', () => {
    const query = new BehaviorSubject('query');
    const streams: Subject<TestItem>[] = [];
    const ranges: ListRange[] = [];
    const failed: boolean[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      pageSize: 1,
      load: (_query, range) => {
        ranges.push(range);
        const stream = new Subject<TestItem>();
        streams.push(stream);
        return stream;
      },
      destroyRef,
    });
    dataSource.appendFailed$.subscribe(value => failed.push(value));

    streams[0].next({id: 'one', value: 'one'});
    streams[0].complete();
    dataSource.loadMore();
    dataSource.loadMore();
    expect(ranges).toHaveLength(2);

    streams[1].error(new Error('append failed'));
    expect(dataSource.snapshot).toEqual([{id: 'one', value: 'one'}]);
    expect(failed.at(-1)).toBe(true);
    dataSource.loadMore();
    expect(ranges).toHaveLength(2);

    dataSource.retry();
    expect(ranges.at(-1)).toEqual({offset: 1, pageSize: 1});
    streams[2].next({id: 'two', value: 'two'});
    streams[2].complete();
    expect(dataSource.snapshot.map(row => row.id)).toEqual(['one', 'two']);
    expect(failed.at(-1)).toBe(false);
    destroyRef.destroy();
  });

  it('loads streamed rows and exposes the current rows to late subscribers', () => {
    const query = new BehaviorSubject('first');
    const rows = new Subject<TestItem>();
    const destroyRef = new TestDestroyRef();
    const loading: boolean[] = [];
    let loadCount = 0;
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: () => {
        loadCount++;
        return rows;
      },
      destroyRef,
    });

    dataSource.loading$.subscribe(value => loading.push(value));
    rows.next({id: 'one', value: 'first'});
    rows.next({id: 'two', value: 'second'});
    rows.complete();

    let connectedRows: readonly TestItem[];
    dataSource.connect().subscribe(value => connectedRows = value);

    expect(connectedRows).toEqual([
      {id: 'one', value: 'first'},
      {id: 'two', value: 'second'},
    ]);
    expect(loading).toEqual([true, false]);
    expect(loadCount).toBe(1);
    destroyRef.destroy();
  });

  it('cancels an obsolete load without clearing the active loading state', () => {
    const query = new Subject<string>();
    const streams = new Map<string, Subject<TestItem>>();
    const cancelled: string[] = [];
    const loading: boolean[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: value => new Observable<TestItem>(subscriber => {
        const stream = new Subject<TestItem>();
        streams.set(value, stream);
        const subscription = stream.subscribe(subscriber);
        return () => {
          cancelled.push(value);
          subscription.unsubscribe();
        };
      }),
      destroyRef,
    });
    dataSource.loading$.subscribe(value => loading.push(value));

    query.next('first');
    streams.get('first').next({id: 'old', value: 'old'});
    query.next('second');

    expect(cancelled).toEqual(['first']);
    expect(dataSource.snapshot).toEqual([]);
    expect(loading).toEqual([false, true]);

    streams.get('second').next({id: 'new', value: 'new'});
    streams.get('second').complete();

    expect(dataSource.snapshot).toEqual([{id: 'new', value: 'new'}]);
    expect(loading).toEqual([false, true, false]);
    destroyRef.destroy();
  });

  it('reloads the latest query and optionally retains existing rows', () => {
    const query = new BehaviorSubject('query');
    const streams: Subject<TestItem>[] = [];
    const resets: number[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: () => {
        const stream = new Subject<TestItem>();
        streams.push(stream);
        return stream;
      },
      destroyRef,
    });
    dataSource.reset$.subscribe(() => resets.push(resets.length));

    streams[0].next({id: 'one', value: 'first'});
    dataSource.reload({retainRows: true});

    expect(streams).toHaveLength(2);
    expect(dataSource.snapshot).toEqual([{id: 'one', value: 'first'}]);
    expect(resets).toHaveLength(0);

    streams[1].next({id: 'two', value: 'second'});
    dataSource.reload();

    expect(streams).toHaveLength(3);
    expect(dataSource.snapshot).toEqual([]);
    expect(resets).toHaveLength(1);
    destroyRef.destroy();
  });

  it('replaces duplicate ids without mutating previously emitted arrays', () => {
    const query = new BehaviorSubject('query');
    const rows = new Subject<TestItem>();
    const emissions: (readonly TestItem[])[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({query$: query, load: () => rows, destroyRef});
    dataSource.connect().subscribe(value => emissions.push(value));

    rows.next({id: 'one', value: 'first'});
    const firstRows = emissions.at(-1);
    rows.next({id: 'one', value: 'updated'});

    expect(firstRows).toEqual([{id: 'one', value: 'first'}]);
    expect(dataSource.snapshot).toEqual([{id: 'one', value: 'updated'}]);
    expect(dataSource.snapshot).not.toBe(firstRows);
    destroyRef.destroy();
  });

  it('ignores null rows and cancels the active load when its owner is destroyed', () => {
    const query = new BehaviorSubject('query');
    const rows = new Subject<TestItem>();
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({query$: query, load: () => rows, destroyRef});

    rows.next(null);
    destroyRef.destroy();

    expect(dataSource.snapshot).toEqual([]);
    expect(rows.observed).toBe(false);
  });

  it('reports completion only for the active request', () => {
    const query = new Subject<string>();
    const streams = new Map<string, Subject<TestItem>>();
    const completions: string[] = [];
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: value => {
        const stream = new Subject<TestItem>();
        streams.set(value, stream);
        return stream;
      },
      destroyRef,
    });
    dataSource.completed$.subscribe(({query: completedQuery}) => completions.push(completedQuery));

    query.next('obsolete');
    streams.get('obsolete').next({id: 'old', value: 'old'});
    query.next('active');
    streams.get('obsolete').complete();

    expect(completions).toEqual([]);

    streams.get('active').next({id: 'new', value: 'new'});
    streams.get('active').complete();

    expect(completions).toEqual(['active']);
    destroyRef.destroy();
  });

  it('upserts live rows within the configured capacity', () => {
    const query = new BehaviorSubject({pageSize: 2});
    const rows = new Subject<TestItem>();
    const destroyRef = new TestDestroyRef();
    const dataSource = ListDataSource.fromQuery({
      query$: query,
      load: () => rows,
      destroyRef,
      capacity: value => value.pageSize,
    });

    rows.next({id: 'one', value: 'first'});
    rows.next({id: 'two', value: 'second'});
    rows.next({id: 'three', value: 'third'});
    rows.next({id: 'one', value: 'updated'});

    expect(dataSource.snapshot).toEqual([
      {id: 'three', value: 'third'},
      {id: 'one', value: 'updated'},
    ]);
    destroyRef.destroy();
  });
});
