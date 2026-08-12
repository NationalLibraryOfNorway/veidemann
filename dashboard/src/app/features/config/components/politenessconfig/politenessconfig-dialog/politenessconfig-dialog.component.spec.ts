import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PolitenessConfigDialogComponent} from './politenessconfig-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';
import {MatCheckboxHarness} from '@angular/material/checkbox/testing';
import {TestbedHarnessEnvironment} from '@angular/cdk/testing/testbed';

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

  it('renders Use hostname as a normal checkbox', async () => {
    const checkbox = await TestbedHarnessEnvironment.loader(fixture).getHarness(MatCheckboxHarness);
    expect(await checkbox.getLabelText()).toBe('Use hostname');
  });
});
