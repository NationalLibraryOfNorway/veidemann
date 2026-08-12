import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CollectionDialogComponent} from './collection-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';

describe('CollectionDialogComponent', () => {
  let component: CollectionDialogComponent;
  let fixture: ComponentFixture<CollectionDialogComponent>;

  const configobject = new ConfigObject({
    kind: Kind.COLLECTION
  });

  const MY_CONF: ConfigDialogData = {
    configObject: configobject,
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [
        CollectionDialogComponent,
      ],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CollectionDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('renders Compressed as a normal checkbox', async () => {
    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    expect(await checkbox.getLabelText()).toBe('Compressed');
  });
});
