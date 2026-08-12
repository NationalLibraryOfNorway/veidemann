import {ComponentFixture, TestBed} from '@angular/core/testing';

import {SeedMultiDialogComponent} from './seed-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind, Meta} from '../../../../../shared/models/config';
import {ConfigDialogData} from '../../../func';
import {By} from '@angular/platform-browser';
import {provideCoreTesting} from '../../../../../core/core.testing.module';


describe('SeedMultiDialogComponent', () => {
  let component: SeedMultiDialogComponent;
  let fixture: ComponentFixture<SeedMultiDialogComponent>;
  const seed = new ConfigObject({
    id: '1000',
    apiVersion: 'v1',
    kind: Kind.SEED,
    meta: new Meta({
      name: 'Test'
    })
  });

  const crawlJob = new ConfigObject({
    id: '1001',
    apiVersion: 'v1',
    kind: Kind.CRAWLJOB,
    meta: new Meta({
      name: 'TestCrawlJob'
    })
  });

  const MY_CONF: ConfigDialogData = {
    configObject: seed,
    options: {crawlJobs: [crawlJob]}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [SeedMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(SeedMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  // TODO: Better tests
  it('should show crawljobs in mat-select', () => {
    const ele = fixture.debugElement.query(By.css('mat-select')).nativeElement;
    // console.log(ele.childNodes[0])
    expect(ele.childNodes[0]).not.toBeNull();
    // expect(ele.childNodes[0].innerHTML).toContain('TestCrawlJob');
  });

  it('should have "options" populated ', () => {
    // console.log(component.data.options);
    expect(component.data.options.crawlJobs).not.toBeNull();
  });

  it('uses concise section headings', () => {
    const sections = fixture.nativeElement.querySelectorAll('.mass-update-section') as NodeListOf<HTMLElement>;
    const headings = Array.from(sections, section =>
      section.querySelector('h5, legend')?.textContent?.trim());

    expect(headings).toEqual(['Labels', 'Crawl jobs', 'Status']);
  });

  it('right-aligns the Crawl jobs toggle and uses a two-column value row', () => {
    const operation = fixture.nativeElement.querySelector('app-multi-update-operation') as HTMLElement;
    const headingRow = operation.querySelector('.heading-row') as HTMLElement;
    const valueRow = operation.querySelector('.value-row') as HTMLElement;
    const heading = headingRow.querySelector('h5') as HTMLElement;
    const toggle = headingRow.querySelector('mat-button-toggle-group') as HTMLElement;
    const commonField = valueRow.querySelector('[multiUpdateCommon]') as HTMLElement;
    const updateField = operation.querySelector('.crawl-job-update-field') as HTMLElement;
    expect(heading.nextElementSibling).toBe(toggle);
    expect(toggle.getBoundingClientRect().right).toBeCloseTo(headingRow.getBoundingClientRect().right, 0);
    expect(toggle.querySelector('.mat-pseudo-checkbox')).toBeNull();
    expect(getComputedStyle(valueRow).display).toBe('grid');
    expect(getComputedStyle(valueRow).gridTemplateColumns).toBe('repeat(2, minmax(0, 1fr))');
    expect(commonField.parentElement?.classList.contains('common-values')).toBe(true);
    expect(updateField.parentElement?.classList.contains('editor')).toBe(true);
    expect(component.updateJobRefListId.disabled).toBe(true);

    (toggle.querySelector('button') as HTMLButtonElement).click();
    fixture.detectChanges();
    expect(component.updateJobRefListId.enabled).toBe(true);
  });

  it('should render single-select status chips with no initial selection', () => {
    const [deactivated, activated] = statusChips();
    expect([activated.textContent.trim(), deactivated.textContent.trim()]).toEqual(['Activated', 'Deactivated']);
    expect([chipSelected(activated), chipSelected(deactivated)])
      .toEqual(['false', 'false']);
    expect(fixture.nativeElement.querySelector('app-boolean-override button[mat-stroked-button]')).toBeNull();
  });

  it('should not derive status selection from common or all-selected values', () => {
    for (const state of [false, true, undefined]) {
      component.configObject.seed.disabled = state;
      component['updateForm']();
      fixture.detectChanges();
      expect(component.disabledSelection).toBeNull();
      expect(statusChips().map(chipSelected)).toEqual(['false', 'false']);
    }

    component.allSelected = true;
    component.configObject.seed.disabled = true;
    component['updateForm']();
    fixture.detectChanges();
    expect(component.disabledSelection).toBeNull();
    expect(statusChips().map(chipSelected)).toEqual(['false', 'false']);
  });

  it('should emit false when Activated is selected', () => {
    statusChips()[1].click();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('seed.disabled');
    expect(result.updateTemplate.seed.disabled).toBe(false);
  });

  it('should emit true when Deactivated is selected', () => {
    statusChips()[0].click();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('seed.disabled');
    expect(result.updateTemplate.seed.disabled).toBe(true);
  });

  it('should omit status after deselecting or reverting', () => {
    const activated = statusChips()[1];
    activated.click();
    activated.click();

    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('seed.disabled');

    activated.click();
    component.onRevert();
    fixture.detectChanges();
    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('seed.disabled');
  });

  function statusChips(): HTMLElement[] {
    return Array.from(fixture.nativeElement.querySelectorAll(
      'app-boolean-override mat-chip-option'
    ) as NodeListOf<HTMLElement>);
  }

  function chipSelected(chip: HTMLElement): string | null {
    return chip.querySelector('[role="option"]')?.getAttribute('aria-selected') ?? null;
  }

});
