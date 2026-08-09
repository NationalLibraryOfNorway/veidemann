import {ComponentFixture, TestBed} from '@angular/core/testing';
import {AbilityServiceSignal} from '@casl/angular';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ExecutionAbortActionComponent} from './execution-abort-action.component';

describe('ExecutionAbortActionComponent', () => {
  let canAbort: boolean;
  let fixture: ComponentFixture<ExecutionAbortActionComponent>;

  beforeEach(async () => {
    canAbort = true;
    await TestBed.configureTestingModule({
      imports: [ExecutionAbortActionComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {
          provide: AbilityServiceSignal,
          useValue: {can: (action: string) => action === 'abort' && canAbort},
        },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ExecutionAbortActionComponent);
    fixture.componentRef.setInput('subject', 'jobexecution');
  });

  it('emits abort when the permitted execution is abortable', () => {
    fixture.componentRef.setInput('abortable', true);
    fixture.detectChanges();
    const abort = vi.fn();
    fixture.componentInstance.abort.subscribe(abort);

    (fixture.nativeElement.querySelector('.abort-action') as HTMLButtonElement).click();

    expect(abort).toHaveBeenCalledOnce();
  });

  it('hides the destructive action when completed or unauthorized', () => {
    fixture.componentRef.setInput('abortable', false);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.destructive-actions')).toBeNull();

    canAbort = false;
    fixture.componentRef.setInput('abortable', true);
    fixture.detectChanges();
    expect(fixture.nativeElement.querySelector('.destructive-actions')).toBeNull();
  });

  it('renders only the button in inline mode', () => {
    fixture.componentRef.setInput('abortable', true);
    fixture.componentRef.setInput('inline', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.abort-action')).not.toBeNull();
    expect(fixture.nativeElement.querySelector('.destructive-actions')).toBeNull();
    expect(fixture.nativeElement.textContent).not.toContain('Crawl actions');
  });
});
