import {CrawlconfigPreviewComponent} from './crawlconfig-preview.component';
import {DurationFormatPipe} from '../../../../../shared/pipes/duration-format.pipe';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {NoopAnimationsModule} from '@angular/platform-browser/animations';
import { provideZonelessChangeDetection } from '@angular/core';


describe('CrawlconfigPreviewComponent', () => {
  let component: CrawlconfigPreviewComponent;
  let fixture: ComponentFixture<CrawlconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [NoopAnimationsModule],
      declarations: [CrawlconfigPreviewComponent, DurationFormatPipe],
      providers: [
        provideZonelessChangeDetection()  // Ensure change detection is zoneless
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlconfigPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLCONFIG});
    await fixture.whenStable();
  });


  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
