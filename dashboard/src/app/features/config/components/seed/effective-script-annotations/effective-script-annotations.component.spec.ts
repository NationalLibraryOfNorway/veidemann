import {ComponentFixture, TestBed} from '@angular/core/testing';
import {AbilityServiceSignal} from '@casl/angular';
import {of, throwError} from 'rxjs';

import {provideMaterialAnimationsDisabled} from '../../../../../core/core.testing.module';
import {Annotation, ConfigObject, Kind, Meta} from '../../../../../shared/models';
import {ConfigService} from '../../../../../shared/services';
import {SCRIPT_ANNOTATION_DRAG_TYPE} from '../../script-annotation-context';
import {EffectiveScriptAnnotationsComponent} from './effective-script-annotations.component';

describe('EffectiveScriptAnnotationsComponent', () => {
  let fixture: ComponentFixture<EffectiveScriptAnnotationsComponent>;
  const getScriptAnnotations = vi.fn(() => of([
    new Annotation({key: 'scope_altSeeds', value: 'https://alt.example'}),
  ]));

  beforeEach(async () => {
    getScriptAnnotations.mockClear();
    await TestBed.configureTestingModule({
      imports: [EffectiveScriptAnnotationsComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: AbilityServiceSignal, useValue: {can: () => true}},
        {provide: ConfigService, useValue: {getScriptAnnotations}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(EffectiveScriptAnnotationsComponent);
    fixture.componentRef.setInput('jobIds', ['job-1', 'job-2']);
    fixture.componentRef.setInput('crawlJobs', [
      new ConfigObject({id: 'job-1', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Daily crawl'})}),
      new ConfigObject({id: 'job-2', kind: Kind.CRAWLJOB, meta: new Meta({name: 'Weekly crawl'})}),
    ]);
    fixture.componentRef.setInput('seedId', 'seed-1');
    fixture.detectChanges();
  });

  it('renders one assist chip per selected crawljob and toggles its effective annotations', () => {
    const chips = fixture.nativeElement.querySelectorAll('.crawl-job-chip') as NodeListOf<HTMLElement>;
    expect(Array.from(chips).map(chip => chip.textContent.replace(/\s+/g, ' ').trim()))
      .toEqual(expect.arrayContaining([
        expect.stringContaining('Daily crawl'),
        expect.stringContaining('Weekly crawl'),
      ]));
    expect(fixture.nativeElement.querySelector('.annotation-panel')).toBeNull();

    chips[0].click();
    fixture.detectChanges();

    expect(getScriptAnnotations).toHaveBeenCalledWith('job-1', 'seed-1');
    expect(chips[0].getAttribute('aria-expanded')).toBe('true');
    expect(fixture.nativeElement.querySelector('.annotation-panel').textContent)
      .toContain('scope_altSeeds');

    chips[0].click();
    fixture.detectChanges();

    expect(chips[0].getAttribute('aria-expanded')).toBe('false');
    expect(fixture.nativeElement.querySelector('.annotation-panel')).toBeNull();
    expect(getScriptAnnotations).toHaveBeenCalledTimes(1);
  });

  it('puts the complete annotation value on the native drag payload', () => {
    const component = fixture.componentInstance;
    const dataTransfer = {
      effectAllowed: 'none',
      setData: vi.fn(),
    };

    component.onAnnotationDragStart(
      {dataTransfer} as unknown as DragEvent,
      new Annotation({key: 'scope_altSeeds', value: 'https://alt.example'}),
    );

    expect(dataTransfer.effectAllowed).toBe('copy');
    expect(dataTransfer.setData).toHaveBeenCalledWith(
      SCRIPT_ANNOTATION_DRAG_TYPE,
      JSON.stringify({key: 'scope_altSeeds', value: 'https://alt.example'}),
    );
    expect(dataTransfer.setData).toHaveBeenCalledWith(
      'text/plain',
      'scope_altSeeds:https://alt.example',
    );
  });

  it('shows a graceful unavailable state when effective annotations cannot be loaded', () => {
    getScriptAnnotations.mockReturnValueOnce(throwError(() => new Error('unavailable')));
    (fixture.nativeElement.querySelector('.crawl-job-chip') as HTMLElement).click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.annotation-unavailable').textContent)
      .toContain('Effective annotations are unavailable for this crawljob.');
  });
});
