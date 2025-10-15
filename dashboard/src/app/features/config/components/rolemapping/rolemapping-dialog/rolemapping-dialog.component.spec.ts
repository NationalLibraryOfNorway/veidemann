import {ComponentFixture, TestBed} from '@angular/core/testing';

import {RoleMappingDialogComponent} from './rolemapping-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('RoleMappingDialogComponent', () => {
  let component: RoleMappingDialogComponent;
  let fixture: ComponentFixture<RoleMappingDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.ROLEMAPPING}),
    options: {},
    allSelected: false
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [RoleMappingDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(RoleMappingDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
