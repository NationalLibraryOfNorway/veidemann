import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BrowserConfigMultiDialogComponent} from './browserconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../shared/models';
import {DurationPickerComponent} from '../..';
import {ConfigDialogData} from '../../../func';
import {LabelMultiComponent} from '../../label/label-multi/label-multi.component';
import {LabelService} from '../../../services';
import {CommonsModule} from '../../../../commons';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {AuthService} from '../../../../core';
import { provideZonelessChangeDetection } from '@angular/core';

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
      imports: [CommonsModule, NoopAnimationsModule],
      declarations: [BrowserConfigMultiDialogComponent, DurationPickerComponent, LabelMultiComponent],
      providers: [
        provideZonelessChangeDetection(),
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
