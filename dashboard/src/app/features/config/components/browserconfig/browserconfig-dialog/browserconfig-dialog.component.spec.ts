import {ComponentFixture, TestBed} from '@angular/core/testing';

import {BrowserConfigDialogComponent} from './browserconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

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
        BrowserConfigDialogComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {
          provide: MatDialogRef,
          useValue: {
            close: () => {
              return;
            }
          }
        },
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
