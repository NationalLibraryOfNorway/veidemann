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
});
