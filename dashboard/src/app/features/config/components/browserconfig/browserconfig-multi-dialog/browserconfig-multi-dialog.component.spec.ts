import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BrowserConfigMultiDialogComponent} from './browserconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {LabelService} from '../../../services';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('BrowserConfigMultiDialogComponent', () => {
  let component: BrowserConfigMultiDialogComponent;
  let fixture: ComponentFixture<BrowserConfigMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject(
      {
        kind: Kind.BROWSERCONFIG
      }),
    options: {},
    allSelected: false
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        BrowserConfigMultiDialogComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: LabelService, useValue: {}},
        {
          provide: AuthService,
          useValue: {
            isAdmin: () => true,
            canUpdate: () => true,
          }
        },
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {
          provide: MatDialogRef,
          useValue: {
            close: () => {
            }
          }
        }]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserConfigMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
