import {ComponentFixture, TestBed} from '@angular/core/testing';
import {CrawljobPreviewComponent} from './crawljob-preview.component';
import {ConfigObject, Kind} from '../../../../../shared/models';
import {provideCoreTesting} from '../../../../../core/core.testing.module';

describe('CrawljobPreviewComponent', () => {
  let component: CrawljobPreviewComponent;
  let fixture: ComponentFixture<CrawljobPreviewComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawljobPreviewComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawljobPreviewComponent);
    component = fixture.componentInstance;
    component.configObject = new ConfigObject({kind: Kind.CRAWLJOB});
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
