import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {DateFnsAdapter, MAT_DATE_FNS_FORMATS} from '@angular/material-date-fns-adapter';
import {DateAdapter, MAT_DATE_FORMATS, MAT_DATE_LOCALE} from '@angular/material/core';
import {MatChipListboxHarness} from '@angular/material/chips/testing';
import {MatDateRangeInputHarness} from '@angular/material/datepicker/testing';
import type {Locale} from 'date-fns';
import {enUS, nb} from 'date-fns/locale';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {JobExecutionState} from '../../../../shared/models';
import {JobExecutionStatusQuery} from '../../services';
import {JobExecutionStatusQueryComponent} from './job-execution-status-query.component';

for (const {name, locale} of [
  {name: 'English', locale: enUS},
  {name: 'Norwegian', locale: nb},
]) {
  describe(`JobExecutionStatusQueryComponent ${name} calendar`, () => {
    let fixture: ComponentFixture<JobExecutionStatusQueryComponent>;
    let loader: HarnessLoader;

    beforeEach(async () => {
      await TestBed.configureTestingModule({
        imports: [JobExecutionStatusQueryComponent],
        providers: [
          provideMaterialAnimationsDisabled(),
          {provide: DateAdapter, useClass: DateFnsAdapter, deps: [MAT_DATE_LOCALE]},
          {provide: MAT_DATE_FORMATS, useValue: MAT_DATE_FNS_FORMATS},
          {provide: MAT_DATE_LOCALE, useValue: locale as Locale},
        ],
      }).compileComponents();

      fixture = TestBed.createComponent(JobExecutionStatusQueryComponent);
      fixture.componentRef.setInput('crawlJobOptions', []);
      fixture.componentRef.setInput('query', {
        stateList: [JobExecutionState.RUNNING],
        jobId: '',
        startTimeFrom: '',
        startTimeTo: '',
        active: '',
        direction: '',
      } satisfies JobExecutionStatusQuery);
      fixture.detectChanges();
      await fixture.whenStable();
      loader = TestbedHarnessEnvironment.loader(fixture);
    });

    it('selects the numeric state supplied by the route query', async () => {
      const listbox = await loader.getHarness(MatChipListboxHarness.with({
        selector: '[formControlName="stateList"]',
      }));
      const chips = await listbox.getChips();
      const labels = await Promise.all(chips.map(chip => chip.getText()));
      const selected = await Promise.all(chips.map(chip => chip.isSelected()));

      expect(await listbox.isMultiple()).toBe(true);
      expect(labels.filter((_, index) => selected[index])).toEqual(['RUNNING']);
    });

    it('places a polling refresh control beside the state filters without a watch filter', () => {
      expect(fixture.nativeElement.querySelector('mat-checkbox')).toBeNull();
      expect(fixture.nativeElement.querySelector('[formcontrolname="watch"]')).toBeNull();
      expect(fixture.nativeElement.querySelector('.report-status-filter-row app-polling-refresh-button'))
        .not.toBeNull();
    });

    it('places the state filters after the other controls', () => {
      const form = fixture.nativeElement.querySelector('.report-filter-form') as HTMLFormElement;

      expect(form.lastElementChild?.classList).toContain('report-status-filter');
    });

    it('renders an inclusive date range with one From and one To input', async () => {
      const range = await loader.getHarness(MatDateRangeInputHarness);
      const startInput = await range.getStartInput();
      const endInput = await range.getEndInput();

      expect(await range.getLabel()).toBe('Job start date');
      expect(await startInput.getPlaceholder()).toBe('From');
      expect(await endInput.getPlaceholder()).toBe('To');
      expect(fixture.nativeElement.querySelectorAll('mat-form-field')).toHaveLength(2);
      expect(fixture.nativeElement.querySelector('mat-timepicker')).toBeNull();
      expect(fixture.nativeElement.querySelector('mat-timepicker-toggle')).toBeNull();
      expect(fixture.nativeElement.textContent).toContain('Both dates are inclusive');
    });

    it('opens with populated date cells', async () => {
      const range = await loader.getHarness(MatDateRangeInputHarness);

      await range.openCalendar();
      const calendar = await range.getCalendar();
      const cells = await calendar.getCells();

      expect(cells.length).toBeGreaterThan(20);
      expect((await Promise.all(cells.map(cell => cell.getText()))).some(text => !!text.trim())).toBe(true);
    });

    it('uses the following midnight as the inclusive To date boundary', () => {
      let emitted: Partial<JobExecutionStatusQuery> | undefined;
      fixture.componentInstance.queryChange.subscribe(query => emitted = query);
      const selectedDate = new Date(2026, 6, 14, 15, 30);

      fixture.componentInstance.form.patchValue({
        startTimeFrom: selectedDate,
        startTimeTo: selectedDate,
      });

      const expectedFrom = new Date(2026, 6, 14);
      const expectedTo = new Date(2026, 6, 15);
      expect(emitted?.startTimeFrom).toBe(expectedFrom.toISOString());
      expect(emitted?.startTimeTo).toBe(expectedTo.toISOString());
    });

    it('displays an upper-bound timestamp as the preceding inclusive date', () => {
      const selectedDate = new Date(2026, 6, 14);
      const upperBoundary = new Date(2026, 6, 15);

      fixture.componentRef.setInput('query', {
        stateList: [JobExecutionState.RUNNING],
        jobId: '',
        startTimeFrom: selectedDate.toISOString(),
        startTimeTo: upperBoundary.toISOString(),
        active: '',
        direction: '',
      } satisfies JobExecutionStatusQuery);
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
      let emitted: Partial<JobExecutionStatusQuery> | undefined;
      fixture.componentInstance.queryChange.subscribe(query => emitted = query);

      fixture.componentInstance.onQuery(fixture.componentInstance.form.value);

      expect(fixture.componentInstance.form.invalid).toBe(true);
      expect(fixture.nativeElement.textContent).toContain('From must be on or before To');
      expect(emitted).toBeUndefined();
    });
  });
}
