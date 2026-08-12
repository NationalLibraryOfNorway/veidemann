import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlJobDialogComponent} from './crawljob-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';
import {MatSlideToggleHarness} from '@angular/material/slide-toggle/testing';

describe('CrawlJobDialogComponent', () => {
  let component: CrawlJobDialogComponent;
  let fixture: ComponentFixture<CrawlJobDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLJOB}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlJobDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlJobDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
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
    expect(component.form.controls['disabled'].value).toBe(true);
    expect(component.form.dirty).toBe(true);
    expect(component.onDialogClose().crawlJob.disabled).toBe(true);
  });
});
