import {ComponentFixture, TestBed} from '@angular/core/testing';

import {CrawlLogQueryComponent} from './crawl-log-query.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('CrawlLogQueryComponent', () => {
  let component: CrawlLogQueryComponent;
  let fixture: ComponentFixture<CrawlLogQueryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [CrawlLogQueryComponent],
      providers: [
        ...provideCoreTesting,
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(CrawlLogQueryComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('does not offer a watch filter', () => {
    expect(fixture.nativeElement.textContent).not.toContain('Watch');
    expect(component.form.contains('watch')).toBe(false);
  });
});
