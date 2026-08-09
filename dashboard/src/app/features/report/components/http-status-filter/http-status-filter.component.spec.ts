import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideCoreTesting} from '../../../../core/core.testing.module';
import {HttpStatusFamily, HttpStatusFilterComponent, httpStatusFamily} from './http-status-filter.component';

describe('HttpStatusFilterComponent', () => {
  let fixture: ComponentFixture<HttpStatusFilterComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [HttpStatusFilterComponent],
      providers: [...provideCoreTesting],
    }).compileComponents();
    fixture = TestBed.createComponent(HttpStatusFilterComponent);
    fixture.detectChanges();
    await fixture.whenStable();
  });

  it('renders one accessible chip for each HTTP status family', () => {
    const chips = [...fixture.nativeElement.querySelectorAll('mat-chip-option')] as HTMLElement[];

    expect(chips.map(chip => chip.textContent?.trim())).toEqual(['1xx', '2xx', '3xx', '4xx', '5xx']);
    const actions = [...fixture.nativeElement.querySelectorAll('mat-chip-option button[role="option"]')] as HTMLElement[];
    expect(actions.map(action => action.getAttribute('aria-label'))).toEqual([
      'Informational responses (100 – 199)',
      'Successful responses (200 – 299)',
      'Redirection messages (300 – 399)',
      'Client error responses (400 – 499)',
      'Server error responses (500 – 599)',
    ]);
  });

  it('maps only standard response codes to a filter family', () => {
    expect([99, 100, 299, 300, 599, 600].map(httpStatusFamily))
      .toEqual([null, 1, 2, 3, 5, null]);
  });

  it('can show only status families present in the current data', () => {
    fixture.componentRef.setInput('families', [2, 4]);
    fixture.detectChanges();

    const chips = [...fixture.nativeElement.querySelectorAll('mat-chip-option')] as HTMLElement[];
    expect(chips.map(chip => chip.textContent?.trim())).toEqual(['2xx', '4xx']);
  });

  it('shows sorted unique exact codes after the available family chips', () => {
    fixture.componentRef.setInput('families', [2]);
    fixture.componentRef.setInput('statusCodes', [201, 200, 200, 99, 600]);
    fixture.detectChanges();

    const chips = [...fixture.nativeElement.querySelectorAll('mat-chip-option')] as HTMLElement[];
    expect(chips.map(chip => chip.textContent?.trim())).toEqual(['2xx', '200', '201']);
    expect(chips[1].querySelector('button')?.getAttribute('aria-label')).toBe('Status code 200');
  });

  it('emits family and exact-code selections independently', () => {
    const families: HttpStatusFamily[][] = [];
    const statusCodes: number[][] = [];
    fixture.componentInstance.valueChange.subscribe(value => families.push(value));
    fixture.componentInstance.exactValueChange.subscribe(value => statusCodes.push(value));

    fixture.componentInstance.onChange([2, 200]);

    expect(families).toEqual([[2]]);
    expect(statusCodes).toEqual([[200]]);
  });
});
