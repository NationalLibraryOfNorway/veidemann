import {ChangeDetectionStrategy, Component, computed, OnDestroy, signal} from '@angular/core';
import {ActivatedRoute, NavigationStart, Router} from '@angular/router';
import {MatDialog} from '@angular/material/dialog';

import {merge, Observable, of, Subject} from 'rxjs';
import {filter, switchMap, takeUntil, tap} from 'rxjs/operators';
import {toObservable, toSignal} from '@angular/core/rxjs-interop';

import {
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
import {AuthService, ControllerApiService, ErrorService, SnackBarService} from '../../../../core';
import {
  BrowserConfigDetailsComponent,
  BrowserScriptDetailsComponent,
  CollectionDetailsComponent,
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
  SeedDetailsComponent
} from '../../components';
import {OptionsService} from '../../services';
import {RunCrawlDialogComponent} from '../../components/run-crawl-dialog/run-crawl-dialog.component';
import {ConfigService} from '../../../../shared/services';
import {configKindFromPath, ConfigDialogData, dialogByKind} from '../../func';
import {RouterExtraService} from '../../services/router-extra.service';
import {AsyncPipe, Location} from '@angular/common';
import {ShortcutComponent} from '../../components/shortcut/shortcut.component';
import {CrawlExecutionStatusPipe, JobExecutionStatusPipe} from '../../pipe';
import {FlexDirective, LayoutDirective} from '@ngbracket/ngx-layout';
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
    CrawlConfigDetailsComponent,
    CrawlExecutionStatusComponent,
    CrawlExecutionStatusPipe,
    CrawlHostGroupConfigDetailsComponent,
    CrawlJobDetailsComponent,
    EntityDetailsComponent,
    FlexDirective,
    JobExecutionStatusPipe,
    JobStatusComponent,
    LayoutDirective,
    PolitenessConfigDetailsComponent,
    ScheduleDetailsComponent,
    SeedDetailsComponent,
    ShortcutComponent,
    RoleMappingDetailsComponent
  ],
  standalone: true
})
export class ConfigurationComponent implements OnDestroy {
  readonly Kind = Kind;

  private ngUnsubscribe = new Subject<void>();

  private configObject: Subject<ConfigObject>;
  configObject$: Observable<ConfigObject>;

  private readonly reload = signal(0);

  options: ConfigOptions;
  options$: Observable<ConfigOptions>;

  constructor(private authService: AuthService,
              private dataService: ConfigService,
              private snackBarService: SnackBarService,
              private errorService: ErrorService,
              private router: Router,
              private dialog: MatDialog,
              private route: ActivatedRoute,
              private optionsService: OptionsService,
              private controllerApiService: ControllerApiService,
              private routerExtraService: RouterExtraService,
              private location: Location) {
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

    this.configObject$ = merge(this.configObject.asObservable(), configObject$);
  }

  get loading$(): Observable<boolean> {
    return this.dataService.loading$;
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
    if (configObject) {

      const data: ConfigDialogData = {configObject, options: this.options};
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
        tap(() => this.dialog.closeAll())
      ).subscribe();

      dialogRef.afterClosed().pipe(
        filter(_ => !!_)
      ).subscribe((config: ConfigObject) => {
        if (config.id) {
          this.onUpdateConfig(config);
        } else {
          this.onSaveConfig(config);
        }
      });
    }
  }

  onClone(configObject: ConfigObject) {
    this.onCreateConfigWithDialog(ConfigObject.clone(configObject));
  }

  onSaveConfig(configObject: ConfigObject) {
    this.dataService.save(configObject)
      .pipe(takeUntil(this.ngUnsubscribe))
      .subscribe(newConfig => {
        this.configObject.next(newConfig);
        this.router.navigate(['../', newConfig.id], {
          relativeTo: this.route,
        })
          .catch(error => this.errorService.dispatch(error));
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
          }).catch(error => this.errorService.dispatch(error));
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
                ).catch(error => this.errorService.dispatch(error));
              } else {
                this.router.navigate(
                  ['report', 'jobexecution', runCrawlReply.jobExecutionId],
                  {queryParams: {watch: true}}
                ).catch(error => this.errorService.dispatch(error));
              }
            });
        }
      });
  }
}
