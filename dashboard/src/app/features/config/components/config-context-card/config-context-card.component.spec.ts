import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {provideRouter} from '@angular/router';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {
  Annotation,
  BrowserScript,
  ConfigObject,
  ConfigRef,
  CrawlJob,
  CrawlConfig,
  CrawlScheduleConfig,
  Kind,
  Label,
  Meta,
} from '../../../../shared/models';
import {ConfigContextCardComponent} from './config-context-card.component';

@Component({
  template: `
    <app-config-context-card [configRef]="ref" [configObject]="object">
      <span cardHeaderHelpers class="projected-helper">Helpers</span>
    </app-config-context-card>
  `,
  imports: [ConfigContextCardComponent],
  standalone: true,
})
class ConfigContextCardHostComponent {
  readonly ref = new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'});
  readonly object = new ConfigObject({
    kind: Kind.COLLECTION,
    id: 'collection-1',
    meta: new Meta({name: 'Collection', description: 'Collection description'}),
  });
}

describe('ConfigContextCardComponent', () => {
  let fixture: ComponentFixture<ConfigContextCardComponent>;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [ConfigContextCardComponent, ConfigContextCardHostComponent],
      providers: [provideMaterialAnimationsDisabled(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigContextCardComponent);
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}));
  });

  it('renders a compact summary and detail link for a related configuration', async () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({
      kind: Kind.CRAWLSCHEDULECONFIG,
      id: 'schedule-1',
    }));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'schedule-1',
      kind: Kind.CRAWLSCHEDULECONFIG,
      meta: new Meta({name: 'Weekdays', description: 'Weekday morning schedule'}),
      crawlScheduleConfig: new CrawlScheduleConfig({cronExpression: '0 8 * * 1-5'}),
    }));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('mat-card-title').textContent.trim()).toBe('Weekdays');
    expect(fixture.nativeElement.querySelector('mat-card-subtitle')).toBeNull();
    const avatar = fixture.nativeElement.querySelector('div[mat-card-avatar]') as HTMLElement;
    expect(avatar).not.toBeNull();
    expect(avatar.querySelector('mat-icon').textContent.trim()).toBe('schedule');
    expect(avatar.querySelector('mat-icon').getAttribute('aria-hidden')).toBe('true');
    expect(fixture.nativeElement.querySelector('mat-card-content').textContent)
      .not.toContain('Weekday morning schedule');
    expect(fixture.nativeElement.textContent).toContain('0 8 * * 1-5');
    expect(fixture.nativeElement.querySelector('mat-card').getAttribute('appearance')).toBe('filled');
    expect(fixture.nativeElement.querySelector('a').getAttribute('href')).toContain('/config/schedule/schedule-1');
  });

  it('identifies an unavailable related configuration without hiding it', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'missing-id'}));
    fixture.componentRef.setInput('unavailable', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('mat-card-title').textContent.trim()).toBe('missing-id');
    expect(fixture.nativeElement.querySelector('mat-card-subtitle')).toBeNull();
    expect(fixture.nativeElement.querySelector('div[mat-card-avatar] mat-icon').textContent.trim())
      .toBe('settings_system_daydream');
    expect(fixture.nativeElement.querySelector('[role="status"]').textContent).toContain('missing-id');
    expect(fixture.nativeElement.querySelector('a')).toBeNull();
  });

  it('leaves related configuration references to the shortcut chips', async () => {
    const configRef = new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'});
    fixture.componentRef.setInput('configRef', configRef);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: configRef.id,
      kind: configRef.kind,
      crawlConfig: new CrawlConfig({
        collectionRef: new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}),
        browserConfigRef: new ConfigRef({kind: Kind.BROWSERCONFIG, id: 'browser-config-1'}),
        politenessRef: new ConfigRef({kind: Kind.POLITENESSCONFIG, id: 'politeness-1'}),
      }),
    }));
    fixture.detectChanges();
    await fixture.whenStable();

    expect(fixture.nativeElement.querySelector('.summary-chips').textContent)
      .not.toContain('collection-1');
    expect(fixture.nativeElement.querySelector('.summary-chips').textContent)
      .not.toContain('browser-config-1');
    expect(fixture.nativeElement.querySelector('.summary-chips').textContent)
      .not.toContain('politeness-1');
    expect(fixture.nativeElement.querySelector('mat-card-actions a').getAttribute('href'))
      .toBe('/config/crawlconfig/crawl-config-1');
  });

  it('does not expose browser script source in the context card', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'script-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
      browserScript: new BrowserScript({script: 'window.secretScriptSource = true;'}),
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Show script');
    expect(fixture.nativeElement.textContent).not.toContain('secretScriptSource');
    expect(fixture.nativeElement.querySelector('details')).toBeNull();
  });

  it('projects helper content into the card header after its title', () => {
    const hostFixture = TestBed.createComponent(ConfigContextCardHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('mat-card-header') as HTMLElement;
    const title = header.querySelector('mat-card-title');
    const helper = header.querySelector('.projected-helper');
    expect(helper).not.toBeNull();
    expect(title.compareDocumentPosition(helper) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('dims a deactivated crawljob and presents its state without a prefix', () => {
    const jobRef = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
    fixture.componentRef.setInput('configRef', jobRef);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: jobRef.id,
      kind: jobRef.kind,
      meta: new Meta({name: 'Daily crawl'}),
      crawlJob: new CrawlJob({disabled: true}),
    }));
    fixture.detectChanges();

    const card = fixture.nativeElement.querySelector('mat-card') as HTMLElement;
    const state = Array.from<HTMLElement>(card.querySelectorAll('mat-chip'))
      .find(chip => chip.getAttribute('aria-label') === 'Crawljob status: Deactivated');
    expect(card.classList).toContain('inactive');
    expect(state.textContent.trim()).toContain('Deactivated');
    expect(state.textContent).not.toContain('Crawljob status:');
  });

  it('renders expandable effective annotations immediately before labels', () => {
    const jobRef = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
    fixture.componentRef.setInput('configRef', jobRef);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: jobRef.id,
      kind: jobRef.kind,
      meta: new Meta({labelList: [new Label({key: 'owner', value: 'archive'})]}),
      crawlJob: new CrawlJob(),
    }));
    fixture.componentRef.setInput('scriptAnnotationContext', {
      jobRef,
      jobName: 'Daily crawl',
      annotations: [new Annotation({key: 'scope', value: 'news'})],
      unavailable: false,
    });
    fixture.componentRef.setInput('canReadAnnotations', true);
    fixture.detectChanges();

    const annotationSection = fixture.nativeElement.querySelector('.effective-annotations') as HTMLElement;
    const labelHeading = fixture.nativeElement.querySelector('.label-heading') as HTMLElement;
    const toggle = annotationSection.querySelector('.annotation-toggle') as HTMLButtonElement;
    expect(annotationSection.compareDocumentPosition(labelHeading) & Node.DOCUMENT_POSITION_FOLLOWING)
      .toBeTruthy();
    expect(toggle.getAttribute('aria-expanded')).toBe('false');

    toggle.click();
    fixture.detectChanges();

    expect(toggle.getAttribute('aria-expanded')).toBe('true');
    expect(annotationSection.textContent).toContain('scope');
    expect(annotationSection.textContent).toContain('news');
  });

  it('renders effective annotation empty and unavailable states', () => {
    const jobRef = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
    fixture.componentRef.setInput('configRef', jobRef);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: jobRef.id,
      kind: jobRef.kind,
      crawlJob: new CrawlJob(),
    }));
    fixture.componentRef.setInput('canReadAnnotations', true);
    fixture.componentRef.setInput('scriptAnnotationContext', {
      jobRef,
      jobName: 'Daily crawl',
      annotations: [],
      unavailable: false,
    });
    fixture.detectChanges();
    (fixture.nativeElement.querySelector('.annotation-toggle') as HTMLButtonElement).click();
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.annotation-empty').textContent)
      .toContain('No script annotations are active.');

    fixture.componentRef.setInput('scriptAnnotationContext', {
      jobRef,
      jobName: 'Daily crawl',
      annotations: [],
      unavailable: true,
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.annotation-unavailable').textContent)
      .toContain('Effective annotations are unavailable for this crawljob.');
  });

  it('hides effective annotations without annotation read permission', () => {
    const jobRef = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
    fixture.componentRef.setInput('configRef', jobRef);
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: jobRef.id,
      kind: jobRef.kind,
      crawlJob: new CrawlJob(),
    }));
    fixture.componentRef.setInput('scriptAnnotationContext', {
      jobRef,
      jobName: 'Daily crawl',
      annotations: [new Annotation({key: 'scope', value: 'news'})],
      unavailable: false,
    });
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.effective-annotations')).toBeNull();
  });
});
