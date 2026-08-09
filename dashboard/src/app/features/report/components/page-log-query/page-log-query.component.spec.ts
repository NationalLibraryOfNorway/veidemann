import {ComponentFixture, TestBed} from '@angular/core/testing';

import {PageLogQueryComponent} from './page-log-query.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';

describe('PageLogQueryComponent', () => {
  let component: PageLogQueryComponent;
  let fixture: ComponentFixture<PageLogQueryComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PageLogQueryComponent],
      providers: [
        ...provideCoreTesting
      ]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogQueryComponent);
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
