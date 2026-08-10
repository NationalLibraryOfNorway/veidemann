import {ComponentFixture, TestBed} from '@angular/core/testing';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ExecutionStatePresentation} from '../../func';
import {
  ExecutionMetadataComponent,
  ExecutionTerminalEvent,
} from './execution-metadata.component';

describe('ExecutionMetadataComponent', () => {
  const running: ExecutionStatePresentation = {
    icon: 'progress_activity',
    label: 'Running',
    tone: 'active',
  };
  let fixture: ComponentFixture<ExecutionMetadataComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ExecutionMetadataComponent],
      providers: [provideMaterialAnimationsDisabled()],
    }).compileComponents();

    fixture = TestBed.createComponent(ExecutionMetadataComponent);
  });

  function render({
    startTime = '',
    endTime = '',
    terminalEvent = null,
    desiredState = null,
  }: {
    startTime?: string;
    endTime?: string;
    terminalEvent?: ExecutionTerminalEvent;
    desiredState?: ExecutionStatePresentation | null;
  } = {}): HTMLElement {
    fixture.componentRef.setInput('state', running);
    fixture.componentRef.setInput('startTime', startTime);
    fixture.componentRef.setInput('endTime', endTime);
    fixture.componentRef.setInput('terminalEvent', terminalEvent);
    fixture.componentRef.setInput('desiredState', desiredState);
    fixture.detectChanges();
    return fixture.nativeElement.querySelector('.execution-metadata') as HTMLElement;
  }

  function text(element: HTMLElement): string {
    return element.textContent.replace(/\s+/g, ' ').trim();
  }

  it('shows active start metadata without an unavailable or end-time placeholder', () => {
    const metadata = render({
      startTime: '2026-08-10T10:19:00.000Z',
      endTime: '2026-08-10T13:57:00.000Z',
    });

    expect(text(metadata)).toContain('Started Aug 10, 2026 at 10:19 AM');
    expect(text(metadata)).not.toContain('1:57 PM');
    expect(text(metadata)).not.toContain('Not available');
  });

  it('uses a compact range for a same-day finished execution', () => {
    const metadata = render({
      startTime: '2026-08-10T10:19:00.000Z',
      endTime: '2026-08-10T13:57:00.000Z',
      terminalEvent: 'finished',
    });

    expect(text(metadata)).toContain('Aug 10, 2026, 10:19 AM → 1:57 PM');
    expect(text(metadata).match(/Aug 10, 2026/g)).toHaveLength(1);
  });

  it('includes both dates in a cross-day finished range', () => {
    const metadata = render({
      startTime: '2026-08-10T23:19:00.000Z',
      endTime: '2026-08-11T01:57:00.000Z',
      terminalEvent: 'finished',
    });

    expect(text(metadata)).toContain('Aug 10, 2026');
    expect(text(metadata)).toContain('Aug 11, 2026');
    expect(text(metadata)).toContain('→');
  });

  it.each([
    ['failed', 'Failed'],
    ['died', 'Died'],
    ['aborted', 'Aborted'],
  ] as const)('labels a %s terminal time without requiring a start time', (terminalEvent, label) => {
    const metadata = render({
      endTime: '2026-08-10T13:57:00.000Z',
      terminalEvent,
    });

    expect(text(metadata)).toContain(`${label} at Aug 10, 2026, 1:57 PM`);
    expect(text(metadata)).not.toContain('Started');
  });

  it('shows only the state when timestamps and desired state are absent', () => {
    const metadata = render();
    expect(metadata.querySelector('.state-badge > span')?.textContent).toBe('Running');
    expect(metadata.querySelector('.lifecycle-time')).toBeNull();
    expect(metadata.querySelector('.desired-state')).toBeNull();
  });

  it('shows a defined desired state after the lifecycle metadata', () => {
    const desiredState: ExecutionStatePresentation = {
      icon: 'error',
      label: 'Aborted',
      tone: 'error',
    };
    const metadata = render({desiredState});

    expect(text(metadata)).toContain('Desired state:');
    expect(metadata.querySelector('.desired-state .state-badge > span')?.textContent).toBe('Aborted');
    expect(metadata.querySelectorAll('.state-badge')).toHaveLength(2);
  });
});
