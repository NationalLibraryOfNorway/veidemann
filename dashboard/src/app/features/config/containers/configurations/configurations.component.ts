import {ChangeDetectionStrategy, Component, computed, DestroyRef, ErrorHandler, OnDestroy, Signal, signal, inject} from '@angular/core';
import {ActivatedRoute, NavigationStart, Params, Router, RouterLink} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';

import {combineLatest, EMPTY, Observable, of, Subject} from 'rxjs';
import {
  distinctUntilChanged,
  filter,
  map,
  startWith,
  switchMap,
  takeUntil,
  tap
} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {ConfigObject, ConfigRef, Kind, Label, ListDataSource, Seed} from '../../../../shared/models';
import {AuthService, ControllerApiService, SnackBarService} from '../../../../core';
import {
  ActiveConfigFilterChip,
  ActiveFilterChipsComponent,
  ConfigListComponent,
  ConfigQueryComponent,
  DeleteDialogComponent,
  DeleteMultiDialogComponent,
  Parcel,
  RunCrawlDialogComponent
} from '../../components';
import {MultiUpdateDialogComponent} from '../../components/multi-update-dialog/multi-update-dialog.component';
import {SortDirection} from '@angular/material/sort';
import {ConfigService} from '../../../../shared/services';
import {ConfigQuery, Sort} from '../../../../shared/func';
import {OptionsService} from '../../services';
import {
  configKindFromPath,
  ConfigPath,
  configQueryFromParamMap,
  ConfigDialogData,
  ConfigOptions,
  dialogByKind,
  equalConfigCountQuery,
  equalConfigQuery,
  parseConfigSearchTerm
} from '../../func';
import {ReferrerError} from '../../../../shared/error';
import {RunCrawlRequest} from '../../../../shared/models/controller/controller.model';
import {AbilityServiceSignal} from '@casl/angular';
import {AsyncPipe} from '@angular/common';
import {MatProgressBar} from '@angular/material/progress-bar';
import {MatListModule} from '@angular/material/list';
import {MatIcon} from '@angular/material/icon';
import {ActionDirective, FilterDirective, ShortcutDirective} from '../../../../shared/directives';
import {MatMenuItem} from '@angular/material/menu';
import {MatTooltip} from '@angular/material/tooltip';
import {
  BrowserConfigNamePipe,
  CollectionNamePipe,
  CrawlConfigNamePipe,
  CrawlScheduleNamePipe,
  PolitenessConfigNamePipe
} from '../../pipe';
import {MatButtonModule} from '@angular/material/button';
import {SeedDialogComponent} from '../../components/seed/seed-dialog/seed-dialog.component';
import {MongoAbility} from '@casl/ability';
import {configKindIcon} from '../../func/config-kind-icon';


@Component({
  selector: 'app-configurations',
  templateUrl: './configurations.component.html',
  styleUrls: ['./configurations.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    ActiveFilterChipsComponent,
    ConfigListComponent,
    ConfigQueryComponent,
    MatListModule,
    MatIcon,
    MatProgressBar,
    FilterDirective,
    RouterLink,
    ActionDirective,
    MatMenuItem,
    MatTooltip,
    CollectionNamePipe,
    ShortcutDirective,
    BrowserConfigNamePipe,
    PolitenessConfigNamePipe,
    CrawlScheduleNamePipe,
    CrawlConfigNamePipe,
    MatButtonModule,
  ],
  standalone: true
})
export class ConfigurationsComponent implements OnDestroy {
  private authService = inject(AuthService);
  private configService = inject(ConfigService);
  private dataService = inject(ConfigService);
  private snackBarService = inject(SnackBarService);
  private errorHandler = inject(ErrorHandler);
  private router = inject(Router);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private controllerApiService = inject(ControllerApiService);
  private optionsService = inject(OptionsService);
  private abilityService = inject<AbilityServiceSignal<MongoAbility>>(AbilityServiceSignal);
  private destroyRef = inject(DestroyRef);

  readonly Kind = Kind;
  readonly ConfigPath = ConfigPath;
  readonly configKindIcon = configKindIcon;
  protected readonly can: AbilityServiceSignal<MongoAbility>['can'];

  readonly totalLength: Signal<number | null>;
  readonly sortDirection: Signal<SortDirection>;
  readonly sortActive: Signal<string>;

  // checked (selected by checkbox) configObjects
  protected selectedConfigs: ConfigObject[];
  readonly selectionMode = signal(false);

  isAllSelected = false;

  private ngUnsubscribe: Subject<void>;

  readonly query: Signal<ConfigQuery>;
  dataSource: ListDataSource<ConfigObject, ConfigQuery>;
  loading$: Observable<boolean>;

  private recount: Subject<void>;

  readonly entityId: Signal<string>;
  readonly entity: Signal<ConfigObject>;
  readonly showCreateButton: Signal<boolean>;

  options: ConfigOptions;
  options$: Observable<ConfigOptions>;

  readonly currentKind: Signal<Kind>;

  constructor() {

    this.options$ = this.optionsService.options$.pipe(
      tap(options => this.options = options)
    );

    this.ngUnsubscribe = new Subject<void>();

    this.selectedConfigs = [];

    this.recount = new Subject();
    this.can = this.abilityService.can;

    const queryParamMap = toSignal(this.route.queryParamMap, {requireSync: true});
    const kindParamMap = toSignal(this.route.parent.paramMap, {requireSync: true});

    this.currentKind = computed(() => configKindFromPath(kindParamMap().get('kind')));
    this.query = computed(
      () => configQueryFromParamMap(this.currentKind(), queryParamMap()),
      {equal: equalConfigQuery}
    );
    this.entityId = computed(() => this.query().entityId);
    this.sortDirection = computed(() => this.query().direction);
    this.sortActive = computed(() => this.query().active);

    const query$ = toObservable(this.query);

    this.dataSource = ListDataSource.fromQuery<ConfigQuery, ConfigObject>({
      query$,
      load: (query, range) => query.kind === Kind.UNDEFINED ? EMPTY : this.configService.search(query, range),
      destroyRef: this.destroyRef,
    });
    this.dataSource.reset$.pipe(
      takeUntil(this.ngUnsubscribe)
    ).subscribe(() => {
      this.onSelectedChange([]);
    });

    this.loading$ = combineLatest([
      this.dataSource.loading$,
      this.configService.loading$,
    ]).pipe(
      map(([listLoading, operationLoading]) => listLoading || operationLoading),
      distinctUntilChanged()
    );

    const totalLength$: Observable<number | null> = combineLatest([
      this.recount.pipe(startWith(null as string)),
      query$.pipe(
        filter(query => query.kind !== Kind.UNDEFINED),
        distinctUntilChanged(equalConfigCountQuery)
      )
    ]).pipe(
      switchMap(([, query]) => this.configService.count(query).pipe(startWith(null))),
    );

    this.totalLength = toSignal(totalLength$, {initialValue: null});

    const entity$ = toObservable(this.entityId).pipe(
      switchMap(id => id
        ? this.configService.get(new ConfigRef({id, kind: Kind.CRAWLENTITY}))
        : of(null)),
      startWith(null as ConfigObject)
    );
    this.entity = toSignal(entity$, {initialValue: null});

    this.showCreateButton = computed(() => {
      switch (this.currentKind()) {
        case Kind.SEED:
          return this.entity() !== null;
        case Kind.UNDEFINED:
          return false;
        default:
          return true;
      }
    });
  }

  ngOnDestroy(): void {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  canRead(configObject: ConfigObject): boolean {
    return this.authService.canRead(configObject.kind);
  }

  canEdit(configObject: ConfigObject): boolean {
    return this.authService.canUpdate(configObject.kind);
  }

  canClone(configObject: ConfigObject): boolean {
    return this.authService.canCreate(configObject.kind);
  }

  canRunCrawl(configObject: ConfigObject): boolean {
    return this.authService.canRunCrawl(configObject.kind);
  }

  getJobRefListQueryParams(configObject: ConfigObject): Params {
    return {crawl_job_id: configObject.seed.jobRefList.map(jobRef => jobRef.id)};
  }

  onCreateConfigWithDialog(configObject?: ConfigObject) {
    if (!configObject) {
      configObject = new ConfigObject({kind: this.currentKind()});
      if (this.entityId()) {
        configObject.seed.entityRef = new ConfigRef({kind: Kind.CRAWLENTITY, id: this.entityId()});
      }
    }
    const data: ConfigDialogData = {configObject, options: this.options};
    const componentType = dialogByKind(configObject.kind);
    const dialogRef = this.dialog.open(componentType, {data});

    let reload = true;
    // if kind is different then configObjects kind we don't want to reload
    if (this.currentKind() !== configObject.kind) {
      reload = false;
    }
    if (configObject.kind === Kind.SEED) {
      const move = (dialogRef.componentInstance as SeedDialogComponent).move.subscribe((parcel: Parcel) => {
        this.onMoveSeed(parcel);
        move.unsubscribe();
      });
    }

    this.router.events.pipe(
      filter(event => event instanceof NavigationStart),
      tap(() => this.dialog.closeAll())
    ).subscribe();

    dialogRef.afterClosed().pipe(
      filter(_ => !!_)
    ).subscribe((config: ConfigObject) => {
      if (Array.isArray(config)) {
        this.onSaveMultiple(config, reload);
      } else {
        if (config.id) {
          this.onUpdateConfig(config);
        } else {
          this.onSaveConfig(config, reload);
        }
      }
    });
  }

  onEdit(configObject: ConfigObject) {
    this.onCreateConfigWithDialog(configObject);
  }

  onShowDetails(configObject: ConfigObject) {
    this.router.navigate([configObject.id], {
      relativeTo: this.route,
    }).catch(error => this.errorHandler.handleError(error));
  }

  onClone(configObject: ConfigObject) {
    this.onCreateConfigWithDialog(ConfigObject.clone(configObject));
  }

  onUpdateConfig(configObject: ConfigObject) {
    this.configService.update(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(() => {
        this.dataSource.reload();
        this.recountIfFiltered();
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.updated:Updated`);
      });
  }

  onSaveConfig(configObject: ConfigObject, reload = true) {
    this.configService.save(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(() => {
        if (reload) {
          this.dataSource.reload();
          this.recount.next();
        }
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.saved:Saved`);
      });
  }

  onSaveMultiple(configObjects: ConfigObject[], reload = true) {
    this.configService.saveMultiple(configObjects)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(saved => {
        if (reload) {
          this.dataSource.reload();
          this.recount.next();
        }
        this.snackBarService.openSnackBar(saved + $localize`:@snackBarMessage.multipleSaved: configurations saved`);
      });
  }

  onQueryChange(value: Partial<ConfigQuery>): void {
    if (this.selectionMode()) {
      return;
    }
    const queryParams = {
      p: null,
      s: null,
      entity_id: value.entityId || null,
      schedule_id: value.scheduleId || null,
      crawl_config_id: value.crawlConfigId || null,
      collection_id: value.collectionId || null,
      browser_config_id: value.browserConfigId || null,
      politeness_id: value.politenessId || null,
      disabled: value.disabled === null ? null : value.disabled,
      script_type: value.browserScriptType ?? null,
      robots_policy: value.robotsPolicy ?? null,
      role: value.role ?? null,
      crawl_job_id: value.crawlJobIdList.length ? value.crawlJobIdList : null,
      script_id: value.scriptIdList.length ? value.scriptIdList : null,
      q: value.term || null
    };

    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {...queryParams, id: null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onFilterByLabel(label: Label): void {
    if (this.selectionMode()) {
      return;
    }
    this.onQueryChange({
      ...this.query(),
      term: `label:${label.key}:${label.value}`,
    });
  }

  onRemoveFilter(chip: ActiveConfigFilterChip): void {
    if (this.selectionMode()) {
      return;
    }
    const query: ConfigQuery = {
      ...this.query(),
      crawlJobIdList: [...this.query().crawlJobIdList],
      scriptIdList: [...this.query().scriptIdList],
    };
    switch (chip.key) {
      case 'entityId':
        query.entityId = null;
        break;
      case 'scriptIdList':
        query.scriptIdList = query.scriptIdList.filter(id => id !== chip.value);
        break;
      case 'crawlJobIdList':
        query.crawlJobIdList = query.crawlJobIdList.filter(id => id !== chip.value);
        break;
      case 'labelSelector':
        query.term = parseConfigSearchTerm(query.term ?? '').name;
        break;
    }
    this.onQueryChange(query);
  }

  onSort(sort: Sort) {
    if (this.selectionMode()) {
      return;
    }
    this.router.navigate([], {
      relativeTo: this.route,
      queryParamsHandling: 'merge',
      queryParams: {p: null, s: null, sort: sort.active && sort.direction ? `${sort.active}:${sort.direction}` : null}
    }).catch(error => this.errorHandler.handleError(error));
  }

  onDisabledFilterChange(disabled: boolean | null): void {
    if (this.selectionMode()) {
      return;
    }
    this.onQueryChange({...this.query(), disabled});
  }

  onSelectAll() {
    this.isAllSelected = true;
  }

  onRowClick(config: ConfigObject) {
    this.router.navigate([config.id], {relativeTo: this.route})
      .catch(error => this.errorHandler.handleError(error));
  }

  onSelectedChange(configs: ConfigObject[]) {
    this.isAllSelected = false;
    this.selectedConfigs = configs;
    this.selectionMode.set(configs.length > 0);
  }

  onFilterByEntityRef(configObject: ConfigObject) {
    this.router.navigate(['../seed'], {
      queryParams: {entity_id: configObject.seed.entityRef.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByScheduleRef(configObject: ConfigObject) {
    this.router.navigate(['../crawljobs'], {
      queryParams: {schedule_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByCrawlConfigRef(configObject: ConfigObject) {
    this.router.navigate(['../crawljobs'], {
      queryParams: {crawl_config_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByCollectionRef(configObject: ConfigObject) {
    this.router.navigate(['../crawlconfig'], {
      queryParams: {collection_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByBrowserConfigRef(configObject: ConfigObject) {
    this.router.navigate(['../crawlconfig'], {
      queryParams: {browser_config_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByPolitenessConfigRef(configObject: ConfigObject) {
    this.router.navigate(['../crawlconfig'], {
      queryParams: {politeness_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByBrowserScriptRef(configObject: ConfigObject) {
    this.router.navigate(['../browserconfig'], {
      queryParams: {script_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onFilterByCrawlJobRef(configObject: ConfigObject) {
    this.router.navigate(['../seed'], {
      queryParams: {crawl_job_id: configObject.id},
      relativeTo: this.route.parent
    });
  }

  onListSeed(configRef: ConfigRef) {
    this.router.navigate(['seed'], {queryParams: {entity_id: configRef.id}, relativeTo: this.route.parent});
  }

  onCreateSeedFromEntity(entity: ConfigObject) {
    const entityRef = ConfigObject.toConfigRef(entity);
    const configObject = new ConfigObject({kind: Kind.SEED, seed: new Seed({entityRef})});

    this.onCreateConfigWithDialog(configObject);
  }

  onMoveSeed(parcel: Parcel) {
    (Array.isArray(parcel.seed)
      ? this.dataService.moveMultiple(parcel.seed, parcel.entityRef)
      : this.dataService.move(parcel.seed, parcel.entityRef))
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(moved => {
        if (this.hasCountAffectingFilter()) {
          this.dataSource.reload();
          this.recount.next();
        }
        this.snackBarService.openSnackBar(moved + $localize`:@snackBarMessage.multipleMoved: configurations moved`);
      });
  }

  onEditSelected() {
    const configObject = ConfigObject.mergeConfigs(this.selectedConfigs);
    const data: ConfigDialogData = {configObject, options: this.options, allSelected: this.isAllSelected};
    const dialogRef = this.dialog.open(MultiUpdateDialogComponent, {
      data,
      width: '720px',
      maxWidth: 'calc(100vw - 32px)',
      maxHeight: 'calc(100dvh - 32px)',
      autoFocus: 'first-tabbable',
      restoreFocus: true,
    });
    dialogRef.afterClosed().pipe(
      filter(_ => !!_)
    ).subscribe(({updateTemplate, pathList}: { updateTemplate: ConfigObject, pathList: string[] }) => {
      this.onUpdateMulti({updateTemplate, pathList});
    });
  }

  onUpdateMulti({updateTemplate, pathList}: { updateTemplate: ConfigObject, pathList: string[] }) {
    if (this.isAllSelected) {
      this.configService.startUpdateWithTemplate(updateTemplate, pathList)
        .pipe(takeUntil(this.ngUnsubscribe))
        .subscribe(taskId => {
          this.snackBarService.openSnackBar(
            $localize`:@@snackBarMessage.backgroundUpdateStarted:Update started in the background. Reload later to see the result. Task ID: ${taskId}:TASK_ID:`);
        });
      return;
    }

    this.configService.updateWithTemplate(
      updateTemplate, pathList, this.selectedConfigs.map(config => config.id))
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(updatedConfigs => {
        this.dataSource.reload();
        this.recountIfFiltered();
        this.snackBarService.openSnackBar(
          updatedConfigs + $localize`:@snackBarMessage.multipleUpdated: configurations updated`);
      });
  }

  onDeleteConfig(configObject: ConfigObject) {
    const dialogRef = this.dialog.open(DeleteDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {configObject},
    });

    dialogRef.afterClosed()
      .pipe(
        filter(result => !!result),
        switchMap(() => this.configService.delete(configObject)),
        filter(deleted => !!deleted)
      )
      .subscribe(() => {
        this.router.navigate([], {
          relativeTo: this.route.parent,
        }).catch(error => this.errorHandler.handleError(error));
        this.dataSource.reload();
        this.recount.next();
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.deleted:Deleted`);
      });
  }

  onDeleteSelectedConfigs() {
    this.onDeleteConfigObjects(this.selectedConfigs);
  }

  onDeleteConfigObjects(configObjects: ConfigObject[]) {
    const dialogRef = this.dialog.open(DeleteMultiDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {numberOfConfigs: configObjects.length}
    });

    dialogRef.afterClosed()
      .pipe(
        filter(_ => _),
        switchMap(() => this.configService.deleteMultiple(configObjects))
      )
      .subscribe(numDeleted => {
        if (configObjects.length !== numDeleted) {
          this.errorHandler.handleError(new ReferrerError({numConfigs: configObjects.length, numDeleted}));
        } else {
          this.snackBarService.openSnackBar(
            numDeleted + $localize`:@snackBarMessage.multipleDeleted: configurations deleted`);
        }
        this.dataSource.reload();
        this.recount.next();
      });
  }

  private recountIfFiltered(): void {
    if (this.hasCountAffectingFilter()) {
      this.recount.next();
    }
  }

  private hasCountAffectingFilter(): boolean {
    const query = this.query();
    return query.disabled !== null
      || query.browserScriptType !== null
      || query.robotsPolicy !== null
      || !!query.entityId
      || !!query.scheduleId
      || !!query.crawlConfigId
      || !!query.collectionId
      || !!query.browserConfigId
      || !!query.politenessId
      || !!query.term
      || query.crawlJobIdList.length > 0
      || query.scriptIdList.length > 0;
  }

  onRunCrawl(configObject: ConfigObject) {
    const crawlJobs = this.options.crawlJobs;
    const dialogRef = this.dialog.open(RunCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {configObject, jobRefId: null, crawlJobs}
    });

    dialogRef.afterClosed()
      .subscribe(result => {
        if (result?.runCrawlRequest) {
          this.controllerApiService.runCrawl(result.runCrawlRequest)
            .subscribe(runCrawlReply => {
              if (configObject.kind === Kind.SEED) {
                this.router.navigate(
                  ['report', 'crawlexecution'],
                  {
                    queryParams: {
                      job_execution_id: runCrawlReply.jobExecutionId,
                      seed_id: configObject.id,
                    }
                  }
                ).catch(error => this.errorHandler.handleError(error));
              } else {
                this.router.navigate(
                  ['report', 'jobexecution', runCrawlReply.jobExecutionId],
                  {queryParams: {watch: true}}
                ).catch(error => this.errorHandler.handleError(error));
              }
            });
        }
      });
  }

  onRunCrawlSelected(configObjects: ConfigObject[]) {
    const crawlJobs = this.options.crawlJobs;
    const dialogRef = this.dialog.open(RunCrawlDialogComponent, {
      disableClose: false,
      autoFocus: true,
      data: {configObject: configObjects[0], jobRefId: null, crawlJobs, numberOfSeeds: configObjects.length}
    });
    dialogRef.afterClosed()
      .subscribe(result => {
        if (result) {
          if (result.crawlMultiple) {
            let started = 0;
            for (const seed of this.selectedConfigs) {
              this.controllerApiService.runCrawl(
                new RunCrawlRequest({seedId: seed.id, jobId: result.runCrawlRequest.jobId})
              ).subscribe(runCrawlReply => {
                started++;
                if (started === this.selectedConfigs.length) {
                  this.router.navigate(['report', 'crawlexecution'],
                    {
                      queryParams: {
                        job_id: result.runCrawlRequest.jobId,
                        job_execution_id: runCrawlReply.jobExecutionId,
                      }
                    }
                  ).catch(error => this.errorHandler.handleError(error));
                }
              });
            }
          } else {
            this.controllerApiService.runCrawl(result.runCrawlRequest)
              .subscribe(runCrawlReply => {
                this.router.navigate(
                  ['report', 'crawlexecution'],
                  {
                    queryParams: {
                      job_execution_id: runCrawlReply.jobExecutionId,
                      seed_id: configObjects[0].id,
                    }
                  }
                ).catch(error => this.errorHandler.handleError(error));
              });
          }
        }
      });
  }
}
