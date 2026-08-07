import {ComponentFixture, TestBed} from '@angular/core/testing';

import {SeedMultiDialogComponent} from './seed-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind, Meta} from '../../../../../shared/models/config';
import {ConfigDialogData} from '../../../func';
import {By} from '@angular/platform-browser';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {HarnessLoader} from '@angular/cdk/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatChipListboxHarness} from '@angular/material/chips/testing';


describe('SeedMultiDialogComponent', () => {
  let component: SeedMultiDialogComponent;
  let fixture: ComponentFixture<SeedMultiDialogComponent>;
  let loader: HarnessLoader;
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
    loader = TestbedHarnessEnvironment.loader(fixture);
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

  it('should render a single-select status list with no initial selection', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="seed-mass-update-status"]'}));
    const chips = await listbox.getChips();

    expect(await listbox.isMultiple()).toBe(false);
    expect(await Promise.all(chips.map(chip => chip.getText()))).toEqual(['Activated', 'Deactivated']);
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
  });

  it('should not derive status selection from common or all-selected values', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="seed-mass-update-status"]'}));
    const chips = await listbox.getChips();

    for (const state of [false, true, undefined]) {
      component.configObject.seed.disabled = state;
      component['updateForm']();
      fixture.detectChanges();
      expect(component.disabledSelection).toBeNull();
      expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
    }

    component.allSelected = true;
    component.configObject.seed.disabled = true;
    component['updateForm']();
    fixture.detectChanges();
    expect(component.disabledSelection).toBeNull();
    expect(await Promise.all(chips.map(chip => chip.isSelected()))).toEqual([false, false]);
  });

  it('should emit false when Activated is selected', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="seed-mass-update-status"]'}));
    const [activated] = await listbox.getChips();
    await activated.select();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('seed.disabled');
    expect(result.updateTemplate.seed.disabled).toBe(false);
  });

  it('should emit true when Deactivated is selected', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="seed-mass-update-status"]'}));
    const [, deactivated] = await listbox.getChips();
    await deactivated.select();

    const result = component.onDialogClose();
    expect(result.pathList).toContain('seed.disabled');
    expect(result.updateTemplate.seed.disabled).toBe(true);
  });

  it('should omit status after deselecting or reverting', async () => {
    const listbox = await loader.getHarness(MatChipListboxHarness.with({selector: '[aria-labelledby="seed-mass-update-status"]'}));
    const [activated] = await listbox.getChips();
    await activated.select();
    await activated.deselect();

    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('seed.disabled');

    await activated.select();
    component.onRevert();
    fixture.detectChanges();
    expect(component.canUpdate).toBe(false);
    expect(component.onDialogClose().pathList).not.toContain('seed.disabled');
  });

});
