import {ComponentFixture, TestBed} from '@angular/core/testing';
import {BrowserConfigMultiDialogComponent} from './browserconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
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
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {
          provide: MatDialogRef,
          useValue: {
            close: () => {
              return;
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

  it('keeps add/remove editors visible, disabled, and resettable', () => {
    const operations = fixture.nativeElement.querySelectorAll(
      'app-multi-update-operation'
    ) as NodeListOf<HTMLElement>;
    expect(operations.length).toBe(2);
    expect(component.scriptRefIdList.disabled).toBe(true);
    expect(component.scriptSelectorList.disabled).toBe(true);

    (operations[0].querySelector('mat-button-toggle button') as HTMLButtonElement).click();
    fixture.detectChanges();
    expect(component.scriptRefIdList.enabled).toBe(true);

    component.onRevert();
    fixture.detectChanges();
    expect(component.shouldAddBrowserScript).toBeUndefined();
    expect(component.scriptRefIdList.disabled).toBe(true);
    expect(operations[0].querySelectorAll('[aria-pressed="true"]').length).toBe(0);
  });
});
