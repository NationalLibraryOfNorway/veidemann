import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawlhostgroupconfigPreviewComponent} from './crawlhostgroupconfig-preview.component';
import {ConfigObject, Kind} from '../../../../shared/models';
import {DurationFormatPipe} from '../../../../commons/pipes/duration-format.pipe';
import {CommonsModule} from '../../../../commons';
import { provideZonelessChangeDetection } from '@angular/core';

describe('CrawlhostgroupconfigPreviewComponent', () => {
  let component: CrawlhostgroupconfigPreviewComponent;
  let fixture: ComponentFixture<CrawlhostgroupconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CommonsModule],
      declarations: [CrawlhostgroupconfigPreviewComponent, DurationFormatPipe],
      providers: [
        provideZonelessChangeDetection()
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlhostgroupconfigPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLHOSTGROUPCONFIG});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
