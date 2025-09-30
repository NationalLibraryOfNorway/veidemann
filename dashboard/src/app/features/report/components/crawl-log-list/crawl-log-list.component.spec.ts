import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogListComponent} from './crawl-log-list.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlLogListComponent', () => {
  let component: CrawlLogListComponent;
  let fixture: ComponentFixture<CrawlLogListComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlLogListComponent],
      providers: [
        ...provideCoreTesting,
      ],
      declarations: []
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogListComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
