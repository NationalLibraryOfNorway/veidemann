import {Type} from '@angular/core';
import {TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';

import {provideCoreTesting} from '../../../core/core.testing.module';
import {ConfigObject, Kind} from '../../../shared/models';
import {ConfigDialogData} from '../func';
import {BrowserConfigMultiDialogComponent} from './browserconfig/browserconfig-multi-dialog/browserconfig-multi-dialog.component';
import {BrowserScriptMultiDialogComponent} from './browserscript/browserscript-multi-dialog/browserscript-multi-dialog.component';
import {CrawlConfigMultiDialogComponent} from './crawlconfig/crawlconfig-multi-dialog/crawlconfig-multi-dialog.component';
import {CrawlHostGroupConfigMultiDialogComponent} from './crawlhostgroupconfig/crawlhostgroupconfig-multi-dialog/crawlhostgroupconfig-multi-dialog.component';
import {CrawlJobMultiDialogComponent} from './crawljobs/crawljob-multi-dialog/crawljobs-multi-dialog.component';
import {EntityMultiDialogComponent} from './entity/entity-multi-dialog/entity-multi-dialog.component';
import {PolitenessConfigMultiDialogComponent} from './politenessconfig/politenessconfig-multi-dialog/politenessconfig-multi-dialog.component';
import {RoleMappingMultiDialogComponent} from './rolemapping/rolemapping-multi-dialog/rolemapping-multi-dialog.component';
import {ScheduleMultiDialogComponent} from './schedule/schedule-multi-dialog/schedule-multi-dialog.component';
import {SeedMultiDialogComponent} from './seed/seed-multi-dialog/seed-multi-dialog.component';

const guidance = 'Only changed settings will be applied. Values left untouched will be preserved.';
const dialogOptions = {
  browserConfigs: [],
  browserScripts: [],
  collections: [],
  crawlConfigs: [],
  crawlJobs: [],
  crawlScheduleConfigs: [],
  politenessConfigs: [],
  robotsPolicies: [],
  roles: [],
  scopeScripts: [],
};

const dialogs: {component: Type<unknown>; kind: Kind; name: string}[] = [
  {component: EntityMultiDialogComponent, kind: Kind.CRAWLENTITY, name: 'entity'},
  {component: SeedMultiDialogComponent, kind: Kind.SEED, name: 'seed'},
  {component: CrawlJobMultiDialogComponent, kind: Kind.CRAWLJOB, name: 'crawl job'},
  {component: CrawlConfigMultiDialogComponent, kind: Kind.CRAWLCONFIG, name: 'crawl configuration'},
  {component: ScheduleMultiDialogComponent, kind: Kind.CRAWLSCHEDULECONFIG, name: 'schedule'},
  {component: BrowserConfigMultiDialogComponent, kind: Kind.BROWSERCONFIG, name: 'browser configuration'},
  {component: BrowserScriptMultiDialogComponent, kind: Kind.BROWSERSCRIPT, name: 'browser script'},
  {component: PolitenessConfigMultiDialogComponent, kind: Kind.POLITENESSCONFIG, name: 'politeness configuration'},
  {
    component: CrawlHostGroupConfigMultiDialogComponent,
    kind: Kind.CRAWLHOSTGROUPCONFIG,
    name: 'crawl host group configuration'
  },
  {component: RoleMappingMultiDialogComponent, kind: Kind.ROLEMAPPING, name: 'role mapping'},
];

describe('Mass-update dialog headers', () => {
  it.each(dialogs)('renders the $name guidance as a fixed header subtitle', async ({component, kind}) => {
    const data: ConfigDialogData = {
      configObject: new ConfigObject({kind}),
      options: dialogOptions,
      allSelected: false,
    };

    await TestBed.configureTestingModule({
      imports: [component],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: {}},
        {provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE]},
        {provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS},
        {provide: MAT_DATE_LOCALE, useValue: nb},
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(component);
    fixture.detectChanges();
    await fixture.whenStable();

    const element = fixture.nativeElement as HTMLElement;
    const header = element.querySelector('.mass-update-header') as HTMLElement;
    const content = element.querySelector('mat-dialog-content') as HTMLElement;
    const subtitle = header.querySelector('p') as HTMLElement;

    expect(header).not.toBeNull();
    expect(header.querySelector('h2')).not.toBeNull();
    expect(subtitle.textContent.replace(/\s+/g, ' ').trim()).toBe(guidance);
    expect(content.contains(header)).toBe(false);
    expect(content.textContent).not.toContain(guidance);
  });
});
