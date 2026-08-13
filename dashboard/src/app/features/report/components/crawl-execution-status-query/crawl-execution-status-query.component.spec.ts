import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {MatDateRangeInputHarness} from '@angular/material/datepicker/testing';
import {enUS} from 'date-fns/locale';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ConfigObject, CrawlExecutionState, Meta} from '../../../../shared/models';
import {CrawlExecutionStatusQuery} from '../../services';
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
      seedId: '',
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

  it('places the has-error filter before the state group and polling control', () => {
    const statusControls = fixture.nativeElement.querySelector('.report-status-controls-row') as HTMLElement;
    const filterChips = fixture.nativeElement.querySelectorAll(
      'mat-chip-listbox:not([formcontrolname="stateList"]) mat-chip-option'
    ) as NodeListOf<HTMLElement>;
    const contextChip = fixture.nativeElement.querySelector('mat-chip:not(mat-chip-option)') as HTMLElement;
    const stateFieldset = statusControls.children[1] as HTMLFieldSetElement;

    expect(fixture.nativeElement.querySelector('mat-checkbox')).toBeNull();
    expect([...filterChips].map(chip => chip.textContent.trim())).toEqual(['HAS ERROR']);
    expect(fixture.nativeElement.querySelector('[formcontrolname="watch"]')).toBeNull();
    expect(statusControls.children[0].getAttribute('formcontrolname')).toBe('hasError');
    expect(stateFieldset.classList).toContain('report-status-filter');
    expect(stateFieldset.querySelector('legend')?.textContent.trim()).toBe('State');
    expect(stateFieldset.querySelector('[formcontrolname="hasError"]')).toBeNull();
    expect(statusControls.children[2].tagName).toBe('APP-POLLING-REFRESH-BUTTON');
    expect(contextChip).toBeNull();
    expect(fixture.nativeElement.querySelectorAll('mat-select')).toHaveLength(1);
  });

  it('emits the has-error filter without changing the selected states', async () => {
    let emitted: Partial<CrawlExecutionStatusQuery> | undefined;
    fixture.componentInstance.queryChange.subscribe(query => emitted = query);
    const listbox = await loader.getHarness(MatChipListboxHarness.with({
      selector: '[formControlName="hasError"]',
    }));
    const [hasErrorChip] = await listbox.getChips();

    await hasErrorChip.select();

    expect(emitted?.hasError).toBe(true);
    expect(emitted?.stateList).toEqual([CrawlExecutionState.FETCHING, CrawlExecutionState.FINISHED]);
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
