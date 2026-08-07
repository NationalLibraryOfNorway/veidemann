import {ComponentFixture, TestBed} from '@angular/core/testing';

import {Annotation, ConfigRef, Kind} from '../../../../shared/models';
import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {ScriptAnnotationsCardComponent} from './script-annotations-card.component';
import {AbilityServiceSignal} from '@casl/angular';

describe('ScriptAnnotationsCardComponent', () => {
  let fixture: ComponentFixture<ScriptAnnotationsCardComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ScriptAnnotationsCardComponent],
      providers: [
        provideMaterialAnimationsDisabled(),
        {provide: AbilityServiceSignal, useValue: {can: () => true}},
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(ScriptAnnotationsCardComponent);
  });

  it('renders effective annotations for a crawljob', () => {
    fixture.componentRef.setInput('contexts', [{
      jobRef: new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}),
      jobName: 'Daily crawl',
      annotations: [
        new Annotation({key: 'scope_altSeeds', value: 'alt.example'}),
        new Annotation({key: 'browser_language', value: ''}),
      ],
      unavailable: false,
    }]);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-title').textContent.trim())
      .toBe('Effective script annotations');
    const tiles = fixture.nativeElement.querySelectorAll('.metric-tile') as NodeListOf<HTMLElement>;
    expect(tiles.length).toBe(2);
    expect(tiles[0].querySelector('dt').textContent).toContain('scope_altSeeds');
    expect(tiles[0].querySelector('dd').textContent).toContain('alt.example');
    expect(tiles[1].querySelector('dt').textContent).toContain('browser_language');
    expect(tiles[1].querySelector('dd').textContent.trim()).toBe('—');
  });

  it('labels each crawljob when showing more than one context', () => {
    fixture.componentRef.setInput('contexts', [
      {
        jobRef: new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}),
        jobName: 'Daily crawl',
        annotations: [],
        unavailable: false,
      },
      {
        jobRef: new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-2'}),
        jobName: 'Weekly crawl',
        annotations: [],
        unavailable: false,
      },
    ]);
    fixture.detectChanges();

    const headings = Array.from(fixture.nativeElement.querySelectorAll('h3') as NodeListOf<HTMLElement>)
      .map(heading => heading.textContent.trim());
    expect(headings).toEqual(['Daily crawl', 'Weekly crawl']);
  });
});
