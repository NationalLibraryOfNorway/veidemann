import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PolitenessConfigDialogComponent} from './politenessconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('PolitenessConfigDialogComponent', () => {
  let component: PolitenessConfigDialogComponent;
  let fixture: ComponentFixture<PolitenessConfigDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({
      kind: Kind.POLITENESSCONFIG
    }),
    options: {}
  };
  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PolitenessConfigDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PolitenessConfigDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
