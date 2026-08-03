import { Directive, OnInit, inject } from '@angular/core';
import {combineLatest, Observable, Subject} from 'rxjs';
import {ActivatedRoute} from '@angular/router';
import {debounceTime, distinctUntilChanged, map, share, startWith} from 'rxjs/operators';
import {ConfigRef, ListItem} from '../../../shared/models';
import {Detail} from '../../../shared/func';
import {Loader} from '../../../shared/services';

interface Getter<T> extends Loader {
  get(query: Detail | ConfigRef): Observable<T>;
}


@Directive()
export abstract class DetailDirective<T extends ListItem> implements OnInit {
  protected route = inject(ActivatedRoute);
  protected abstract service: Getter<T>;

  protected query$: Observable<Detail>;

  protected reload = new Subject<void>();
  protected reload$ = this.reload.asObservable();

  item$: Observable<T>;

  ngOnInit(): void {
    const routeParam$ = combineLatest([this.route.paramMap, this.route.queryParamMap]).pipe(
      debounceTime(0), // synchronize
      map(([paramMap, queryParamMap]) => ({
        id: paramMap.get('id'),
        watch: queryParamMap.get('watch'),
      })),
      share(),
    );

    const id$: Observable<string> = routeParam$.pipe(
      map(({id}) => id),
      distinctUntilChanged());

    const watch$: Observable<boolean> = routeParam$.pipe(
      map(({watch}) => watch === 'true'),
      distinctUntilChanged());

    this.query$ = combineLatest([id$, watch$, this.reload$.pipe(startWith(null as string))])
      .pipe(
        map(([id, watch]) => ({id, watch})),
        share(),
      );
  }
}
