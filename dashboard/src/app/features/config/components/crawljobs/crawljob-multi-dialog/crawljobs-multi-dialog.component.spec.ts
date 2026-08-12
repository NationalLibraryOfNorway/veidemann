import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlJobMultiDialogComponent} from './crawljobs-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlJobMultiDialogComponent', () => {
  let component: CrawlJobMultiDialogComponent;
  let fixture: ComponentFixture<CrawlJobMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLJOB}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlJobMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlJobMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
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
      component.configObject.crawlJob.disabled = state;
      component['updateForm']();
      fixture.detectChanges();
      expect(component.disabledSelection).toBeNull();
      expect(statusChips().map(chipSelected)).toEqual(['false', 'false']);
    }

    component.allSelected = true;
    component.configObject.crawlJob.disabled = true;
    component['updateForm']();
    fixture.detectChanges();
    expect(component.disabledSelection).toBeNull();
    expect(statusChips().map(chipSelected)).toEqual(['false', 'false']);
  });

  it('should emit false when Activated is selected', () => {
    statusChips()[1].click();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('crawlJob.disabled');
    expect(result.updateTemplate.crawlJob.disabled).toBe(false);
  });

  it('should emit true when Deactivated is selected', () => {
    statusChips()[0].click();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('crawlJob.disabled');
    expect(result.updateTemplate.crawlJob.disabled).toBe(true);
  });

  it('should omit status after deselecting or reverting', () => {
    const activated = statusChips()[1];
    activated.click();
    activated.click();

    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('crawlJob.disabled');

    activated.click();
    component.onRevert();
    fixture.detectChanges();
    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('crawlJob.disabled');
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
