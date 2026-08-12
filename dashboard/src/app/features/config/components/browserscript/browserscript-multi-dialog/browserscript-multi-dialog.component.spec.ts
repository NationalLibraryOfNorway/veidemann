import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BrowserScriptMultiDialogComponent} from './browserscript-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('BrowserScriptMultiDialogComponent', () => {
  let component: BrowserScriptMultiDialogComponent;
  let fixture: ComponentFixture<BrowserScriptMultiDialogComponent>;

  const MY_CONF = {
    configObject: new ConfigObject({kind: Kind.BROWSERSCRIPT})
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [BrowserScriptMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserScriptMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('applies Material styling to every dialog action', () => {
    const buttons = fixture.nativeElement.querySelectorAll('mat-dialog-actions button') as NodeListOf<HTMLButtonElement>;
    expect(buttons.length).toBe(3);
    expect(buttons[0].classList.contains('mat-mdc-button')).toBe(true);
    expect(buttons[1].classList.contains('mat-mdc-button')).toBe(true);
    expect(buttons[2].classList.contains('mat-mdc-unelevated-button')).toBe(true);
  });
});
