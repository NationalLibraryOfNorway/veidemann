import {ChangeDetectionStrategy, Component, computed, ErrorHandler, OnDestroy, signal, inject} from '@angular/core';
import {ActivatedRoute, NavigationStart, Router} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';

import {combineLatest, forkJoin, merge, Observable, of, Subject} from 'rxjs';
import {catchError, filter, map, shareReplay, switchMap, take, takeUntil, tap} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {
  Annotation,
  BrowserScriptType,
  ConfigObject,
  ConfigRef,
  Kind,
  RobotsPolicy,
  Role,
  RotationPolicy,
  Seed,
  SubCollectionType
} from '../../../../shared/models';
import {AuthService, ControllerApiService, SnackBarService} from '../../../../core';
import {
  BrowserConfigDetailsComponent,
  BrowserScriptDetailsComponent,
  CollectionDetailsComponent,
  ConfigContextCardComponent,
  CrawlConfigDetailsComponent,
  CrawlExecutionStatusComponent,
  CrawlHostGroupConfigDetailsComponent,
  CrawlJobDetailsComponent,
  DeleteDialogComponent,
  EntityDetailsComponent,
  JobStatusComponent,
  Parcel,
  PolitenessConfigDetailsComponent,
  RoleMappingDetailsComponent,
  ScheduleDetailsComponent,
  ScriptAnnotationContext,
  ScriptAnnotationsCardComponent,
  SeedDetailsComponent
} from '../../components';
import {OptionsResolver, OptionsService} from '../../services';
import {RunCrawlDialogComponent} from '../../components/run-crawl-dialog/run-crawl-dialog.component';
import {ConfigService} from '../../../../shared/services';
import {configKindFromPath, ConfigDialogData, ConfigPath, dialogByKind, relatedConfigRefs} from '../../func';
import {RouterExtraService} from '../../services/router-extra.service';
import {AsyncPipe, Location} from '@angular/common';
import {ConfigShortcutHelpersComponent} from '../../components/shortcut/shortcut.component';
import {CrawlExecutionStatusPipe, JobExecutionStatusPipe} from '../../pipe';
import {SeedDialogComponent} from '../../components/seed/seed-dialog/seed-dialog.component';


export interface ConfigOptions {
  rotationPolicies?: RotationPolicy[];
  subCollectionTypes?: SubCollectionType[];
  crawlConfigs?: ConfigObject[];
  crawlScheduleConfigs?: ConfigObject[];
  browserConfigs?: ConfigObject[];
  collections?: ConfigObject[];
  politenessConfigs?: ConfigObject[];
  browserScripts?: ConfigObject[];
  scopeScripts?: ConfigObject[];
  browserScriptTypes?: BrowserScriptType[];
  robotsPolicies?: RobotsPolicy[];
  crawlJobs?: ConfigObject[];
  roles?: Role[];
}

export interface RelatedConfigContext {
  ref: ConfigRef;
  configObject: ConfigObject | null;
  unavailable: boolean;
}

@Component({
  selector: 'app-configuration',
  templateUrl: './configuration.component.html',
  styleUrls: ['./configuration.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    AsyncPipe,
    BrowserScriptDetailsComponent,
    BrowserConfigDetailsComponent,
    CollectionDetailsComponent,
    ConfigContextCardComponent,
    CrawlConfigDetailsComponent,
    CrawlExecutionStatusComponent,
    CrawlExecutionStatusPipe,
    CrawlHostGroupConfigDetailsComponent,
    CrawlJobDetailsComponent,
    EntityDetailsComponent,
    JobExecutionStatusPipe,
    JobStatusComponent,
    PolitenessConfigDetailsComponent,
    ScheduleDetailsComponent,
    SeedDetailsComponent,
    ConfigShortcutHelpersComponent,
    RoleMappingDetailsComponent,
    ScriptAnnotationsCardComponent,
  ],
  standalone: true
})
export class ConfigurationComponent implements OnDestroy {
  private authService = inject(AuthService);
  private dataService = inject(ConfigService);
  private snackBarService = inject(SnackBarService);
  private errorHandler = inject(ErrorHandler);
  private router = inject(Router);
  private dialog = inject(MatDialog);
  private route = inject(ActivatedRoute);
  private optionsService = inject(OptionsService);
  private optionsResolver = inject(OptionsResolver);
  private controllerApiService = inject(ControllerApiService);
  private routerExtraService = inject(RouterExtraService);
  private location = inject(Location);

  readonly Kind = Kind;

  private ngUnsubscribe = new Subject<void>();

  private configObject: Subject<ConfigObject>;
  configObject$: Observable<ConfigObject>;
  relatedConfigs$: Observable<RelatedConfigContext[]>;
  scriptAnnotationContexts$: Observable<ScriptAnnotationContext[]>;
  annotationSuggestions$: Observable<string[]>;

  private readonly reload = signal(0);

  options: ConfigOptions;
  options$: Observable<ConfigOptions>;

  constructor() {
    this.configObject = new Subject();

    this.options$ = this.optionsService.options$.pipe(
      tap(options => {
        this.options = options;
      }),
    );

    const paramMap = toSignal(this.route.paramMap, {requireSync: true});
    const parentParamMap = toSignal(this.route.parent.paramMap, {requireSync: true});
    const configRef = computed(() => {
      this.reload();
      return new ConfigRef({
        kind: configKindFromPath(parentParamMap().get('kind')),
        id: paramMap().get('id'),
      });
    });

    const configObject$: Observable<ConfigObject> = toObservable(configRef).pipe(
      switchMap(configRef =>
        configRef && configRef.id ? this.dataService.get(configRef) : of(null))
    );

    this.configObject$ = merge(this.configObject.asObservable(), configObject$).pipe(
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.relatedConfigs$ = combineLatest([this.configObject$, this.options$]).pipe(
      switchMap(([configObject, options]) => {
        const refs = relatedConfigRefs(configObject, options?.browserScripts)
          .filter(ref => this.authService.canRead(ref.kind));
        if (!refs.length) {
          return of([] as RelatedConfigContext[]);
        }
        return forkJoin(refs.map(ref => this.dataService.get(ref).pipe(
          map(related => ({ref, configObject: related, unavailable: false} as RelatedConfigContext)),
          catchError(() => of({ref, configObject: null, unavailable: true} as RelatedConfigContext)),
        )));
      }),
    );

    this.scriptAnnotationContexts$ = combineLatest([this.configObject$, this.options$]).pipe(
      switchMap(([configObject, options]) => this.loadScriptAnnotationContexts(configObject, options)),
      shareReplay({bufferSize: 1, refCount: true}),
    );
    this.annotationSuggestions$ = this.scriptAnnotationContexts$.pipe(
      map(contexts => Array.from(new Set(
        contexts.flatMap(context => context.annotations.map(annotation => annotation.key))
      )).sort((a, b) => a.localeCompare(b))),
    );
  }

  get loading$(): Observable<boolean> {
    return this.dataService.loading$;
  }

  private loadScriptAnnotationContexts(
    configObject: ConfigObject,
    options: ConfigOptions | null,
  ): Observable<ScriptAnnotationContext[]> {
    const jobs = this.scriptAnnotationJobs(configObject, options);
    if (!jobs.length) {
      return of([]);
    }

    const seedId = configObject.kind === Kind.SEED ? configObject.id : undefined;
    return forkJoin(jobs.map(job => this.dataService.getScriptAnnotations(job.ref.id, seedId).pipe(
      map((annotations: Annotation[]) => ({
        jobRef: job.ref,
        jobName: job.name,
        annotations,
        unavailable: false,
      } as ScriptAnnotationContext)),
      catchError(() => of({
        jobRef: job.ref,
        jobName: job.name,
        annotations: [],
        unavailable: true,
      } as ScriptAnnotationContext)),
    )));
  }

  private scriptAnnotationJobs(
    configObject: ConfigObject,
    options: ConfigOptions | null,
  ): {ref: ConfigRef; name: string}[] {
    let jobs: {ref: ConfigRef; name: string}[] = [];

    switch (configObject?.kind) {
      case Kind.CRAWLJOB:
        if (configObject.id) {
          jobs = [{
            ref: ConfigObject.toConfigRef(configObject),
            name: configObject.meta.name || configObject.id,
          }];
        }
        break;
      case Kind.SEED:
        jobs = (configObject.seed?.jobRefList ?? []).map(ref => ({
          ref,
          name: options?.crawlJobs?.find(job => job.id === ref.id)?.meta.name || ref.id,
        }));
        break;
      case Kind.CRAWLENTITY:
        jobs = (options?.crawlJobs ?? []).map(job => ({
          ref: ConfigObject.toConfigRef(job),
          name: job.meta.name || job.id,
        }));
        break;
    }

    const seen = new Set<string>();
    return jobs.filter(job => {
      if (!job.ref?.id || seen.has(job.ref.id)) {
        return false;
      }
      seen.add(job.ref.id);
      return true;
    });
  }

  ngOnDestroy(): void {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  onCreateSeedFromEntity(entity: ConfigObject) {
    const entityRef = ConfigObject.toConfigRef(entity);
    const configObject = new ConfigObject({kind: Kind.SEED, seed: new Seed({entityRef})});

    this.onCreateConfigWithDialog(configObject);
  }

  onCreateConfigWithDialog(configObject: ConfigObject) {
    if (!configObject) {
      return;
    }

    this.optionsResolver.load(configObject.kind).pipe(
      take(1),
      takeUntil(this.ngUnsubscribe),
    ).subscribe({
      next: kindOptions => {
        const options = {...this.options, ...kindOptions};
        this.optionsService.next(kindOptions);
        this.openConfigDialog(configObject, options);
      },
      error: error => this.errorHandler.handleError(error),
    });
  }

  private openConfigDialog(configObject: ConfigObject, options: ConfigOptions) {
    const data: ConfigDialogData = {configObject, options};
    const componentType = dialogByKind(configObject.kind);
    const dialogRef = this.dialog.open(componentType, {data});

    if (configObject.kind === Kind.SEED) {
      const move = (dialogRef.componentInstance as SeedDialogComponent).move.subscribe((parcel: Parcel) => {
        this.onMoveSeed(parcel);
        move.unsubscribe();
      });
    }

    this.router.events.pipe(
      filter(event => event instanceof NavigationStart),
      tap(() => this.dialog.closeAll()),
      takeUntil(this.ngUnsubscribe),
    ).subscribe();

    dialogRef.afterClosed().pipe(
      filter(_ => !!_),
      takeUntil(this.ngUnsubscribe),
    ).subscribe((config: ConfigObject) => {
      if (config.id) {
        this.onUpdateConfig(config);
      } else {
        this.onSaveConfig(config);
      }
    });
  }

  onClone(configObject: ConfigObject) {
    this.onCreateConfigWithDialog(ConfigObject.clone(configObject));
  }

  onSaveConfig(configObject: ConfigObject) {
    this.dataService.save(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(newConfig => {
        this.configObject.next(newConfig);
        this.router.navigate(['/config', ConfigPath[newConfig.kind], newConfig.id])
          .catch(error => this.errorHandler.handleError(error));
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.saved:Saved`);
      });
  }

  onUpdateConfig(configObject: ConfigObject) {
    this.dataService.update(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(newConfig => {
        this.configObject.next(newConfig);
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.updated:Updated`);
      });
  }


  onDeleteConfig(configObject: ConfigObject) {
    const dialogRef = this.dialog.open(DeleteDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {configObject},
    });
    const previousUrl = this.routerExtraService.getPreviousUrl();
    const currentUrl = this.routerExtraService.getCurrentUrl();
    dialogRef.afterClosed()
      .pipe(
        filter(result => !!result),
        switchMap(() => this.dataService.delete(configObject)),
        filter(deleted => !!deleted)
      )
      .subscribe(() => {
        if (previousUrl !== currentUrl) {
          this.location.back();
        } else {
          this.router.navigate(['../'], {
            relativeTo: this.route,
          }).catch(error => this.errorHandler.handleError(error));
        }
        this.snackBarService.openSnackBar($localize`:@snackBarMessage.deleted:Deleted`);
      });
  }

  onSaveMultipleSeeds(configObjects: ConfigObject[]) {
    this.dataService.saveMultiple(configObjects)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(saved => {
        this.snackBarService.openSnackBar(saved + $localize`:@snackBarMessage.multipleSaved: configurations saved`);
        this.configObject.next(null);
        this.reload.update(value => value + 1);
      });
  }

  onMoveSeed(parcel: Parcel) {
    (Array.isArray(parcel.seed)
      ? this.dataService.moveMultiple(parcel.seed, parcel.entityRef)
      : this.dataService.move(parcel.seed, parcel.entityRef))
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(moved => {
        if (moved > 0) {
          this.reload.update(value => value + 1);
        }
        this.snackBarService.openSnackBar(moved + $localize`:@snackBarMessage.multipleMoved: configurations moved`);
      });
  }

  onRunCrawl(configObject: ConfigObject) {
    const crawlJobs = this.options.crawlJobs;
    const dialogRef = this.dialog.open(RunCrawlDialogComponent, {
      disableClose: true,
      autoFocus: true,
      data: {configObject, jobRefId: null, crawlJobs}
    });
    dialogRef.afterClosed()
      .subscribe(result => {
        if (result.runCrawlRequest) {
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
}
