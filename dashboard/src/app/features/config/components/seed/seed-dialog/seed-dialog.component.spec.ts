import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {MatSlideToggleHarness} from '@angular/material/slide-toggle/testing';

import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {ConfigObject, ConfigRef, Kind, Seed} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {SeedDialogComponent} from './seed-dialog.component';

describe('SeedDialogComponent', () => {
  let fixture: ComponentFixture<SeedDialogComponent>;

  const data: ConfigDialogData = {
    configObject: new ConfigObject({
      kind: Kind.SEED,
      seed: new Seed({entityRef: new ConfigRef({kind: Kind.CRAWLENTITY, id: 'entity-1'})}),
    }),
    options: {crawlJobs: []},
  };

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [SeedDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: data},
        {provide: MatDialogRef, useValue: {}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(SeedDialogComponent);
  });

  it('renders an inverted, unlabeled Active toggle last in the tab order', async () => {
    fixture.detectChanges();
    const toggle = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatSlideToggleHarness);
    const toggleElement = fixture.nativeElement.querySelector('.configuration-active-toggle') as HTMLElement;
    const cancelButton = fixture.nativeElement.querySelector('button[mat-dialog-close]') as HTMLButtonElement;

    expect(await toggle.getLabelText()).toBe('');
    expect(await toggle.getAriaLabel()).toBe('Active');
    expect(await toggle.isChecked()).toBe(true);
    expect(cancelButton.compareDocumentPosition(toggleElement) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    await toggle.uncheck();
    expect(fixture.componentInstance.form.controls['disabled'].value).toBe(true);
    expect(fixture.componentInstance.form.dirty).toBe(true);
    expect((fixture.componentInstance.onDialogClose() as ConfigObject).seed.disabled).toBe(true);
  });
});
