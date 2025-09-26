import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserConfigDialogComponent} from './browserconfig-dialog.component';
import {ReactiveFormsModule} from '@angular/forms';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {DatePipe} from '@angular/common';
import {ConfigDialogData} from '../../../func';
import {LabelService} from '../../../services';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import {of} from 'rxjs';
import {AuthService} from '../../../../../core';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatInputModule} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';

describe('BrowserConfigDialogComponent', () => {
  let component: BrowserConfigDialogComponent;
  let fixture: ComponentFixture<BrowserConfigDialogComponent>;


  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({
      kind: Kind.BROWSERCONFIG,
    }),
    options: {},
    allSelected: false
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        NoopAnimationsModule,
        BrowserConfigDialogComponent,
        MatButtonModule,
        MatDialogModule,
        MatInputModule,
        ReactiveFormsModule
      ],
      providers: [
        ...provideCoreTesting,
        {
          provide: LabelService,
          useValue: {
            getLabelKeys: () => of([])
          }
        },
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
        },
        DatePipe,
      ],
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(BrowserConfigDialogComponent);
    component = fixture.componentInstance;

    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
