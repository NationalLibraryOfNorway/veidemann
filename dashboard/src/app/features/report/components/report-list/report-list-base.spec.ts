import {ElementRef} from '@angular/core';
import {Sort} from '@angular/material/sort';
import {Subject} from 'rxjs';

import {ListDataSource, ListItem} from '../../../../shared/models';
import {ReportListBaseComponent} from './report-list-base';

interface TestRow extends ListItem {
  value: string;
}

class TestReportListComponent extends ReportListBaseComponent<TestRow> {}

describe('ReportListBaseComponent', () => {
  let component: TestReportListComponent;
  let completed: Subject<unknown>;
  let retry: ReturnType<typeof vi.fn>;
  let loadMore: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    component = new TestReportListComponent();
    completed = new Subject();
    retry = vi.fn();
    loadMore = vi.fn();
    component.dataSource = {
      completed$: completed,
      retry,
      loadMore,
    } as unknown as ListDataSource<TestRow, unknown>;
  });

  afterEach(() => {
    component.ngOnDestroy();
    vi.unstubAllGlobals();
  });

  it('emits sorting and row activation without selection state', () => {
    const row = {id: 'one', value: 'One'};
    const sorts: Sort[] = [];
    const rows: TestRow[] = [];
    component.sort.subscribe(value => sorts.push(value));
    component.rowClick.subscribe(value => rows.push(value));

    component.onSortChange({active: 'value', direction: 'desc'});

    const rowElement = document.createElement('div');
    const button = document.createElement('button');
    rowElement.append(button);
    rowElement.addEventListener('click', event => component.onRowClick(row, event));
    rowElement.addEventListener('keydown', event => component.onRowKeydown(row, event));
    rowElement.dispatchEvent(new MouseEvent('click', {bubbles: true}));
    rowElement.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    button.dispatchEvent(new MouseEvent('click', {bubbles: true}));
    button.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));

    expect(sorts).toEqual([{active: 'value', direction: 'desc'}]);
    expect(rows).toEqual([row, row]);
    expect('selection' in component).toBe(false);
    expect('selectedChange' in component).toBe(false);
    expect('selectAll' in component).toBe(false);
  });

  it('activates primary links without allowing native or row activation', () => {
    const row = {id: 'one', value: 'One'};
    const rows: TestRow[] = [];
    component.rowClick.subscribe(value => rows.push(value));
    const container = document.createElement('div');
    const link = document.createElement('a');
    container.append(link);
    container.addEventListener('click', event => component.onRowClick(row, event));
    link.addEventListener('click', event => component.onPrimaryLink(row, event));

    const event = new MouseEvent('click', {bubbles: true, cancelable: true});
    link.dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
    expect(rows).toEqual([row]);
  });

  it('retries failed loads and loads more when the sentinel intersects', () => {
    let intersectionCallback!: IntersectionObserverCallback;
    const observe = vi.fn();
    const disconnect = vi.fn();
    class TestIntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback;
      }

      observe = observe;
      disconnect = disconnect;
      unobserve = vi.fn();
      takeRecords = vi.fn(() => []);
      root = null;
      rootMargin = '';
      thresholds = [];
    }
    vi.stubGlobal('IntersectionObserver', TestIntersectionObserver);

    component.retry();
    const sentinel = document.createElement('div');
    component.loadMoreSentinel = new ElementRef(sentinel);
    intersectionCallback([
      {isIntersecting: false} as IntersectionObserverEntry,
      {isIntersecting: true} as IntersectionObserverEntry,
    ], {} as IntersectionObserver);

    expect(retry).toHaveBeenCalledTimes(1);
    expect(observe).toHaveBeenCalledWith(sentinel);
    expect(loadMore).toHaveBeenCalledTimes(1);

    component.ngOnDestroy();
    expect(disconnect).toHaveBeenCalled();
  });

  it('does not load more while client-side filters are active', () => {
    let intersectionCallback!: IntersectionObserverCallback;
    class TestIntersectionObserver {
      constructor(callback: IntersectionObserverCallback) {
        intersectionCallback = callback;
      }

      observe = vi.fn();
      disconnect = vi.fn();
      unobserve = vi.fn();
      takeRecords = vi.fn(() => []);
      root = null;
      rootMargin = '';
      thresholds = [];
    }
    vi.stubGlobal('IntersectionObserver', TestIntersectionObserver);

    component.loadMoreDisabled = true;
    component.loadMoreSentinel = new ElementRef(document.createElement('div'));
    intersectionCallback([
      {isIntersecting: true} as IntersectionObserverEntry,
    ], {} as IntersectionObserver);

    expect(loadMore).not.toHaveBeenCalled();
  });
});
