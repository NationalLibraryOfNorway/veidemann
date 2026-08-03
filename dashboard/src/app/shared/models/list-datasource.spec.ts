import {DestroyRef} from '@angular/core';
import {BehaviorSubject, Observable, Subject} from 'rxjs';

import {ListDataSource, ListItem} from './list-datasource';

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
