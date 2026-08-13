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

  it('keeps only a hidden execution-ID control', () => {
    const executionInput = fixture.nativeElement.querySelector('[formcontrolname="executionId"]') as HTMLElement;

    expect(component.form.contains('executionId')).toBe(true);
    expect(component.form.contains('jobExecutionId')).toBe(false);
    expect(component.form.contains('uri')).toBe(false);
    const executionField = executionInput.closest('mat-form-field') as HTMLElement;
    expect(executionField.hidden).toBe(true);
    expect(getComputedStyle(executionField).display).toBe('none');
  });
});
