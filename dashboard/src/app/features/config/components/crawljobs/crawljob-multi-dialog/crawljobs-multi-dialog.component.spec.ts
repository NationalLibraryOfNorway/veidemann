import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlJobMultiDialogComponent} from './crawljobs-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatChipListboxHarness} from '@angular/material/chips/testing';

describe('CrawlJobMultiDialogComponent', () => {
  let component: CrawlJobMultiDialogComponent;
  let fixture: ComponentFixture<CrawlJobMultiDialogComponent>;
  let loader: HarnessLoader;

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
    loader = TestbedHarnessEnvironment.loader(fixture);
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should render a single-select status list with no initial selection', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="crawl-job-mass-update-status"]'}));
    const chips = await listbox.getChips();

    expect(await listbox.isMultiple()).toBe(false);
    expect(await Promise.all(chips.map(chip => chip.getText()))).toEqual(['Activated', 'Deactivated']);
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
  });

  it('should not derive status selection from common or all-selected values', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="crawl-job-mass-update-status"]'}));
    const chips = await listbox.getChips();

    for (const state of [false, true, undefined]) {
      component.configObject.crawlJob.disabled = state;
      component['updateForm']();
      fixture.detectChanges();
      expect(component.disabledSelection).toBeNull();
      expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
    }

    component.allSelected = true;
    component.configObject.crawlJob.disabled = true;
    component['updateForm']();
    fixture.detectChanges();
    expect(component.disabledSelection).toBeNull();
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
  });

  it('should emit false when Activated is selected', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="crawl-job-mass-update-status"]'}));
    const [activated] = await listbox.getChips();
    await activated.select();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('crawlJob.disabled');
    expect(result.updateTemplate.crawlJob.disabled).toBe(false);
  });

  it('should emit true when Deactivated is selected', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="crawl-job-mass-update-status"]'}));
    const [, deactivated] = await listbox.getChips();
    await deactivated.select();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('crawlJob.disabled');
    expect(result.updateTemplate.crawlJob.disabled).toBe(true);
  });

  it('should omit status after deselecting or reverting', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="crawl-job-mass-update-status"]'}));
    const [activated] = await listbox.getChips();
    await activated.select();
    await activated.deselect();

    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('crawlJob.disabled');

    await activated.select();
    component.onRevert();
    fixture.detectChanges();
    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('crawlJob.disabled');
  });
});
