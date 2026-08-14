import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {MatButtonHarness} from '@angular/material/button/testing';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {MatDateRangeInputHarness} from '@angular/material/datepicker/testing';
import {enUS} from 'date-fns/locale';
import {of} from 'rxjs';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigObject, CrawlExecutionState, JobExecutionStatus, Kind, Meta} from '../../../../shared/models';
import {CrawlExecutionService, CrawlExecutionStatusQuery, JobExecutionService} from '../../services';
import {CrawlExecutionStatusQueryComponent} from './crawl-execution-status-query.component';

describe('CrawlExecutionStatusQueryComponent', () => {
  let fixture: ComponentFixture<CrawlExecutionStatusQueryComponent>;
  let loader: HarnessLoader;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [CrawlExecutionStatusQueryComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE]},
        {provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS},
        {provide: MAT_DATE_LOCALE, useValue: enUS},
        {provide: JobExecutionService, useValue: {
          get: ({id}: {id: string}) => of(new JobExecutionStatus({id, jobId: 'job-1'})),
          getJob: () => of(new ConfigObject({
            id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily crawl'}),
          })),
        }},
        {provide: CrawlExecutionService, useValue: {
          getSeed: () => of(new ConfigObject({
            id: 'seed-1', kind: Kind.SEED, meta: new Meta({name: 'Example seed'}),
          })),
        }},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(CrawlExecutionStatusQueryComponent);
    fixture.componentRef.setInput('crawlJobOptions', [
      new ConfigObject({id: 'job-1', meta: new Meta({name: 'Daily crawl'})})
    ]);
    fixture.componentRef.setInput('query', {
      stateList: [CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED],
      jobId: 'job-1',
      jobExecutionId: 'execution-1',
      seedId: 'seed-1',
      startTimeFrom: '',
      startTimeTo: '',
      hasError: false,
      active: '',
      direction: '',
    } satisfies CrawlExecutionStatusQuery);
    fixture.detectChanges();
    await fixture.whenStable();
    loader = TestbedHarnessEnvironment.loader(fixture);
  });

  it('selects all numeric states supplied by the route query', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({
      selector: '[formControlName="stateList"]',
    }));
    const chips = await listbox.getChips();
    const labels = await Promise.all(chips.map(chip => chip.getText()));
    const selected = await Promise.all(chips.map(chip => chip.isSelected()));

    expect(await listbox.isMultiple()).toBe(true);
    expect(labels).toEqual([
      'CREATED',
      'FETCHING',
      'SLEEPING',
      'FINISHED',
      'ABORTED_TIMEOUT',
      'ABORTED_SIZE',
      'ABORTED_MANUAL',
      'FAILED',
      'DIED',
    ]);
    expect(labels.filter((_, index) => selected[index])).toEqual(['FETCHING', 'FINISHED']);
  });

  it('places the compact has-error button after the state group and before polling', async () => {
    const statusControls = fixture.nativeElement.querySelector('.report-status-controls-row') as HTMLElement;
    const stateFieldset = statusControls.children[0] as HTMLFieldSetElement;
    const hasError = statusControls.children[1] as HTMLButtonElement;

    expect(fixture.nativeElement.querySelector('mat-checkbox')).toBeNull();
    expect(fixture.nativeElement.querySelector('[formcontrolname="hasError"]')).toBeNull();
    expect(fixture.nativeElement.querySelector('[formcontrolname="watch"]')).toBeNull();
    expect(stateFieldset.classList).toContain('report-status-filter');
    expect(stateFieldset.querySelector('legend')?.textContent.trim()).toBe('State');
    expect(hasError.classList).toContain('has-error-filter');
    expect(hasError.querySelector('mat-icon')?.textContent.trim()).toBe('error');
    expect(hasError.getAttribute('aria-label')).toBe('Has error');
    expect(hasError.getAttribute('mattooltip')).toBe('Has error');
    expect(hasError.getAttribute('aria-pressed')).toBe('false');
    expect(hasError.classList).toContain('stateful-filter-button');
    expect(hasError.classList).toContain('compact-icon-button');
    const hasErrorHarness = await loader.getHarness(MatButtonHarness.with({selector: '.has-error-filter'}));
    expect(await hasErrorHarness.getAppearance()).toBe('text');
    expect(getComputedStyle(hasError.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
    expect(statusControls.children[2].tagName).toBe('APP-POLLING-REFRESH-BUTTON');
    expect(fixture.nativeElement.querySelectorAll('mat-select')).toHaveLength(1);
  });

  it('hides direct-ID fields and resolves prefix-free chips with representative icons', () => {
    const directInputs = [...fixture.nativeElement.querySelectorAll(
      '[formcontrolname="jobExecutionId"], [formcontrolname="seedId"]'
    )] as HTMLInputElement[];
    const chips = [...fixture.nativeElement.querySelectorAll('.report-active-filter-chips mat-chip')] as HTMLElement[];

    expect(directInputs).toHaveLength(2);
    expect(directInputs.every(input => (input.closest('mat-form-field') as HTMLElement).hidden)).toBe(true);
    expect(directInputs.every(input => getComputedStyle(input.closest('mat-form-field') as HTMLElement).display === 'none'))
      .toBe(true);
    expect(chips.map(chip => chip.textContent.trim())).toEqual([
      expect.stringContaining('Daily crawl'),
      expect.stringContaining('Example seed'),
    ]);
    expect(chips[0].textContent).not.toContain('Job execution:');
    expect(chips[1].textContent).not.toContain('Seed:');
    expect(chips.map(chip => chip.querySelector('mat-icon[matChipAvatar]')?.textContent.trim()))
      .toEqual(['hdr_strong', 'link']);
    expect(chips[0].querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove job execution execution-1 filter');
    expect(chips[1].querySelector('button[matChipRemove]')?.getAttribute('aria-label'))
      .toBe('Remove seed seed-1 filter');

    const form = fixture.nativeElement.querySelector('.report-filter-form') as HTMLElement;
    const dateField = form.querySelector('mat-date-range-input')?.closest('mat-form-field') as HTMLElement;
    const chipSet = form.querySelector('.report-active-filter-chips') as HTMLElement;
    const statusRow = form.querySelector('.report-status-controls-row') as HTMLElement;
    expect(dateField.compareDocumentPosition(chipSet) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(chipSet.compareDocumentPosition(statusRow) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(getComputedStyle(chipSet).flexBasis).toBe('100%');
  });

  it('removes one direct-ID filter without changing the other query values', () => {
    let emitted: Partial<CrawlExecutionStatusQuery> | undefined;
    fixture.componentInstance.queryChange.subscribe(query => emitted = query);

    fixture.componentInstance.removeDirectFilter('jobExecutionId');

    expect(emitted?.jobExecutionId).toBe('');
    expect(emitted?.seedId).toBe('seed-1');
    expect(emitted?.stateList).toEqual([CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED]);
  });

  it('toggles the has-error filter without changing the selected states', () => {
    let emitted: Partial<CrawlExecutionStatusQuery> | undefined;
    fixture.componentInstance.queryChange.subscribe(query => emitted = query);
    const hasError = fixture.nativeElement.querySelector('.has-error-filter') as HTMLButtonElement;
    const inactiveColor = getComputedStyle(hasError).color;

    hasError.click();
    fixture.detectChanges();

    expect(emitted?.hasError).toBe(true);
    expect(emitted?.stateList).toEqual([CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED]);
    expect(hasError.getAttribute('aria-pressed')).toBe('true');
    expect(getComputedStyle(hasError).color).not.toBe(inactiveColor);
    expect(getComputedStyle(hasError.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 1');

    hasError.click();
    fixture.detectChanges();

    expect(emitted?.hasError).toBe(false);
    expect(emitted?.stateList).toEqual([CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED]);
    expect(hasError.getAttribute('aria-pressed')).toBe('false');
    expect(getComputedStyle(hasError).color).toBe(inactiveColor);
    expect(getComputedStyle(hasError.querySelector('mat-icon') as HTMLElement).fontVariationSettings)
      .toContain('"FILL" 0');
  });

  it('initializes the has-error button from the route query', () => {
    fixture.componentRef.setInput('query', {
      stateList: [CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED],
      jobId: 'job-1',
      jobExecutionId: 'execution-1',
      seedId: 'seed-1',
      startTimeFrom: '',
      startTimeTo: '',
      hasError: true,
      active: '',
      direction: '',
    } satisfies CrawlExecutionStatusQuery);
    fixture.detectChanges();

    const hasError = fixture.nativeElement.querySelector('.has-error-filter') as HTMLButtonElement;
    expect(hasError.getAttribute('aria-pressed')).toBe('true');
  });

  it('places the state filters after the other controls', () => {
    const form = fixture.nativeElement.querySelector('.report-filter-form') as HTMLFormElement;
    const statusControls = form.lastElementChild as HTMLElement;

    expect(statusControls.classList).toContain('report-status-controls-row');
    expect(statusControls.querySelector('.report-status-filter')).not.toBeNull();
  });

  it('renders an inclusive date range without time inputs', async () => {
    const range = await loader.getHarness(MatDateRangeInputHarness);
    const startInput = await range.getStartInput();
    const endInput = await range.getEndInput();

    expect(await range.getLabel()).toBe('Crawl start date');
    expect(await startInput.getPlaceholder()).toBe('From');
    expect(await endInput.getPlaceholder()).toBe('To');
    expect(fixture.nativeElement.querySelectorAll('mat-form-field')).toHaveLength(4);
    expect(fixture.nativeElement.querySelector('mat-timepicker')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-timepicker-toggle')).toBeNull();
    expect(fixture.nativeElement.textContent).toContain('Both dates are inclusive');
  });

  it('uses the following midnight as the inclusive To date boundary', () => {
    let emitted: Partial<CrawlExecutionStatusQuery> | undefined;
    fixture.componentInstance.queryChange.subscribe(query => emitted = query);
    const selectedDate = new Date(2026, 6, 14, 15, 30);

    fixture.componentInstance.form.patchValue({
      startTimeFrom: selectedDate,
      startTimeTo: selectedDate,
    });

    expect(emitted?.startTimeFrom).toBe(new Date(2026, 6, 14).toISOString());
    expect(emitted?.startTimeTo).toBe(new Date(2026, 6, 15).toISOString());
  });

  it('displays an upper-bound timestamp as the preceding inclusive date', () => {
    const selectedDate = new Date(2026, 6, 14);
    const upperBoundary = new Date(2026, 6, 15);

    fixture.componentRef.setInput('query', {
      stateList: [CrawlExecutionState.FETCHING],
      jobId: '',
      jobExecutionId: '',
      seedId: '',
      startTimeFrom: selectedDate.toISOString(),
      startTimeTo: upperBoundary.toISOString(),
      hasError: false,
      active: '',
      direction: '',
    } satisfies CrawlExecutionStatusQuery);
    fixture.detectChanges();

    const formFrom = fixture.componentInstance.form.controls['startTimeFrom'].value as Date;
    const formTo = fixture.componentInstance.form.controls['startTimeTo'].value as Date;
    expect([formFrom.getFullYear(), formFrom.getMonth(), formFrom.getDate()]).toEqual([2026, 6, 14]);
    expect([formTo.getFullYear(), formTo.getMonth(), formTo.getDate()]).toEqual([2026, 6, 14]);
  });

  it('does not emit a reversed date range', () => {
    fixture.componentInstance.form.patchValue({
      startTimeFrom: new Date(2026, 6, 15),
      startTimeTo: new Date(2026, 6, 14),
    });
    fixture.componentInstance.form.markAllAsTouched();
    fixture.detectChanges();
    let emitted: Partial<CrawlExecutionStatusQuery> | undefined;
    fixture.componentInstance.queryChange.subscribe(query => emitted = query);

    fixture.componentInstance.onQuery(fixture.componentInstance.form.value);

    expect(fixture.componentInstance.form.invalid).toBe(true);
    expect(fixture.nativeElement.textContent).toContain('From must be on or before To');
    expect(emitted).toBeUndefined();
  });
});
