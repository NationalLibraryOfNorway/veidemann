import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PolitenessConfigMultiDialogComponent} from './politenessconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('PolitenessConfigMultiDialogComponent', () => {
  let component: PolitenessConfigMultiDialogComponent;
  let fixture: ComponentFixture<PolitenessConfigMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({
      kind: Kind.POLITENESSCONFIG
    }),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PolitenessConfigMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}}
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PolitenessConfigMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('uses explicit hostname choices and clears a repeated selection', () => {
    const [useHostname] = fixture.nativeElement.querySelectorAll(
      'app-boolean-override button'
    ) as NodeListOf<HTMLButtonElement>;

    useHostname.click();
    let result = component.onDialogClose();
    expect(result.pathList).toContain('politenessConfig.useHostname');
    expect(result.updateTemplate.politenessConfig.useHostname).toBe(true);

    useHostname.click();
    result = component.onDialogClose();
    expect(result.pathList).not.toContain('politenessConfig.useHostname');
  });
});
