import {ComponentFixture, TestBed} from '@angular/core/testing';

import {ExecutionStatePresentation} from '../../func';
import {ExecutionMetadataComponent} from './execution-metadata.component';

describe('ExecutionMetadataComponent', () => {
  const running: ExecutionStatePresentation = {
    icon: 'progress_activity', label: 'Running', tone: 'active', lifecycle: 'active',
  };
  let fixture: ComponentFixture<ExecutionMetadataComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({imports: [ExecutionMetadataComponent]}).compileComponents();
    fixture = TestBed.createComponent(ExecutionMetadataComponent);
  });

  function render({
    state = running, startTime = '', endTime = '', desiredState = null,
  }: {
    state?: ExecutionStatePresentation;
    startTime?: string;
    endTime?: string;
    desiredState?: ExecutionStatePresentation | null;
  } = {}): HTMLElement {
    fixture.componentRef.setInput('state', state);
    fixture.componentRef.setInput('startTime', startTime);
    fixture.componentRef.setInput('endTime', endTime);
    fixture.componentRef.setInput('desiredState', desiredState);
    fixture.detectChanges();
    return fixture.nativeElement as HTMLElement;
  }

  function text(element: HTMLElement): string {
    return element.textContent?.replace(/\s+/g, ' ').trim() ?? '';
  }

  it('renders an active execution as Started and Running through Now', () => {
    const metadata = render({startTime: '2026-08-10T10:19:00.000Z'});
    const terms = [...metadata.querySelectorAll('dt')].map(term => term.textContent?.trim());
    expect(terms).toEqual(['Started', 'Running']);
    expect(text(metadata)).toContain('Aug 10, 2026, 10:19:00 AM');
    expect(text(metadata)).toContain('Now');
  });

  it('places optional context after lifecycle timestamps in a wrapping three-column grid', () => {
    fixture.componentRef.setInput('contextLabel', 'Crawl job');
    fixture.componentRef.setInput('contextValue', 'Daily crawl');
    const metadata = render({startTime: '2026-08-10T10:19:00.000Z'});
    const lifecycle = metadata.querySelector('.execution-lifecycle') as HTMLElement;
    const terms = [...metadata.querySelectorAll('dt')].map(term => term.textContent?.trim());

    expect(terms).toEqual(['Started', 'Running', 'Crawl job']);
    expect(metadata.querySelectorAll('dd')[2].textContent?.trim()).toBe('Daily crawl');
    expect(lifecycle.classList).toContain('execution-lifecycle--with-context');
  });

  it('offers a full-width metric grid with compact lifecycle values and a row divider', () => {
    fixture.componentRef.setInput('presentation', 'metrics');
    fixture.componentRef.setInput('contextLabel', 'Crawl job');
    fixture.componentRef.setInput('contextValue', 'Daily crawl');
    const metadata = render({startTime: '2026-08-10T10:19:00.000Z'});
    const lifecycle = metadata.querySelector('.execution-lifecycle') as HTMLElement;

    expect(lifecycle.classList).toContain('execution-lifecycle--metrics');
    expect(getComputedStyle(lifecycle).maxWidth).toBe('none');
    expect(getComputedStyle(lifecycle).borderBottomStyle).toBe('solid');
    expect(getComputedStyle(lifecycle).paddingBottom).toBe('20px');
  });

  it.each([
    ['Finished', 'terminal'],
    ['Failed', 'terminal'],
    ['Died', 'terminal'],
    ['Aborted manually', 'terminal'],
    ['Aborted after timeout', 'terminal'],
    ['Aborted at size limit', 'terminal'],
  ] as const)('uses the terminal verb %s with the localized end time', (label, lifecycle) => {
    const state: ExecutionStatePresentation = {icon: 'error', label, tone: 'error', lifecycle};
    const metadata = render({state, endTime: '2026-08-10T13:57:00.000Z'});
    expect(metadata.querySelectorAll('dt')[1].textContent?.trim()).toBe(label);
    expect(text(metadata)).toContain('Aug 10, 2026, 1:57:00 PM');
    expect(text(metadata)).not.toContain('Now');
  });

  it('uses missing-value text for absent start and terminal timestamps', () => {
    const failed: ExecutionStatePresentation = {
      icon: 'error', label: 'Failed', tone: 'error', lifecycle: 'terminal',
    };
    const metadata = render({state: failed});
    expect(text(metadata).match(/Not available/g)).toHaveLength(2);
  });

  it('uses Ended for an undefined terminal state', () => {
    const ended: ExecutionStatePresentation = {
      icon: 'help', label: 'Ended', tone: 'neutral', lifecycle: 'undefined',
    };
    const metadata = render({state: ended});
    expect(metadata.querySelectorAll('dt')[1].textContent?.trim()).toBe('Ended');
  });

  it('shows a distinct desired state in place of Now', () => {
    const aborted: ExecutionStatePresentation = {
      icon: 'error', label: 'Aborted manually', tone: 'error', lifecycle: 'terminal',
    };
    let metadata = render({desiredState: aborted});
    expect(text(metadata)).toContain('Running Aborted manually');
    expect(text(metadata)).not.toContain('Now');
    expect(text(metadata)).not.toContain('Requested:');

    metadata = render({desiredState: running});
    expect(text(metadata)).toContain('Now');
    expect(text(metadata)).not.toContain('Aborted manually');
  });

  it('suppresses a desired state equal to the completed state when an end time exists', () => {
    const aborted: ExecutionStatePresentation = {
      icon: 'error', label: 'Aborted manually', tone: 'error', lifecycle: 'terminal',
    };
    const metadata = render({
      state: aborted,
      desiredState: aborted,
      endTime: '2026-08-10T13:57:00.000Z',
    });

    expect(metadata.querySelectorAll('dt')[1].textContent?.trim()).toBe('Aborted manually');
    expect(text(metadata.querySelectorAll('dd')[1])).toBe('Aug 10, 2026, 1:57:00 PM');
    expect(text(metadata).match(/Aborted manually/g)).toHaveLength(1);
  });
});
