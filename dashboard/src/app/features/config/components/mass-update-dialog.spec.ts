import {TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {nb} from 'date-fns/locale';

import {provideCoreTesting} from '../../../core/core.testing.module';
import {ConfigObject, Kind} from '../../../shared/models';
import {ConfigDialogData} from '../func';
import {MultiUpdateDialogComponent} from './multi-update-dialog/multi-update-dialog.component';

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
  rotationPolicies: [],
  subCollectionTypes: [],
};

const dialogs: {kind: Kind; name: string}[] = [
  {kind: Kind.CRAWLENTITY, name: 'entity'},
  {kind: Kind.SEED, name: 'seed'},
  {kind: Kind.CRAWLJOB, name: 'crawl job'},
  {kind: Kind.CRAWLCONFIG, name: 'crawl configuration'},
  {kind: Kind.CRAWLSCHEDULECONFIG, name: 'schedule'},
  {kind: Kind.BROWSERCONFIG, name: 'browser configuration'},
  {kind: Kind.BROWSERSCRIPT, name: 'browser script'},
  {kind: Kind.POLITENESSCONFIG, name: 'politeness configuration'},
  {kind: Kind.CRAWLHOSTGROUPCONFIG, name: 'crawl host group configuration'},
  {kind: Kind.ROLEMAPPING, name: 'role mapping'},
  {kind: Kind.COLLECTION, name: 'collection'},
];

describe('MultiUpdateDialogComponent', () => {
  it.each(dialogs)('renders the $name through the shared shell', async ({kind}) => {
    const data: ConfigDialogData = {
      configObject: new ConfigObject({kind}),
      options: dialogOptions,
      allSelected: false,
    };

    await TestBed.configureTestingModule({
      imports: [MultiUpdateDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: {}},
        {provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE]},
        {provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS},
        {provide: MAT_DATE_LOCALE, useValue: nb},
      ]
    }).compileComponents();

    const fixture = TestBed.createComponent(MultiUpdateDialogComponent);
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
    const sectionHost = element.querySelector('.mass-update-section-host') as HTMLElement;
    const embeddedHeader = sectionHost.querySelector('.mass-update-header') as HTMLElement;
    expect(sectionHost).not.toBeNull();
    expect(getComputedStyle(embeddedHeader).display).toBe('none');
  });
});
