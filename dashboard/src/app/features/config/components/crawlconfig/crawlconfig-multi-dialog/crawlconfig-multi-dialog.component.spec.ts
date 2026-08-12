import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlConfigMultiDialogComponent} from './crawlconfig-multi-dialog.component';
import {MAT_DIALOG_DATA, MatDialogRef} from '@angular/material/dialog';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ConfigDialogData} from '../../../func';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlConfigMultiDialogComponent', () => {
  let component: CrawlConfigMultiDialogComponent;
  let fixture: ComponentFixture<CrawlConfigMultiDialogComponent>;

  const MY_CONF: ConfigDialogData = {
    configObject: new ConfigObject({kind: Kind.CRAWLCONFIG}),
    options: {}
  };

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlConfigMultiDialogComponent],
      providers: [
        ...provideCoreTesting,
        {provide: MAT_DIALOG_DATA, useValue: MY_CONF},
        {provide: MatDialogRef, useValue: {}},
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlConfigMultiDialogComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('uses explicit screenshot choices and clears a repeated selection', () => {
    const [takeScreenshot] = fixture.nativeElement.querySelectorAll(
      'app-boolean-override button'
    ) as NodeListOf<HTMLButtonElement>;

    takeScreenshot.click();
    let result = component.onDialogClose();
    expect(result.pathList).toContain('crawlConfig.extra.createScreenshot');
    expect(result.updateTemplate.crawlConfig.extra.createScreenshot).toBe(true);

    takeScreenshot.click();
    result = component.onDialogClose();
    expect(result.pathList).not.toContain('crawlConfig.extra.createScreenshot');
  });
});
