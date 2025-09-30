import {CrawlconfigPreviewComponent} from './crawlconfig-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {ComponentFixture, TestBed} from '@angular/core/testing';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawlconfigPreviewComponent', () => {
  let component: CrawlconfigPreviewComponent;
  let fixture: ComponentFixture<CrawlconfigPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlconfigPreviewComponent],
      providers: [
        ...provideCoreTesting,
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
