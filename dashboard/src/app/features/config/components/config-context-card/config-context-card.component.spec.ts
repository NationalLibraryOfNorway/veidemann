import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {provideRouter} from '@angular/router';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {
  BrowserScript,
  ConfigObject,
  ConfigRef,
  CrawlJob,
  CrawlConfig,
  CrawlScheduleConfig,
  Kind,
  Meta,
} from '../../../../shared/models';
import {ConfigContextCardComponent} from './config-context-card.component';

@Component({
  template: `
    <app-config-context-card [configRef]="ref" [configObject]="object">
      <span relationLinks class="projected-helper">Helpers</span>
      <span itemMenu class="projected-menu">Menu</span>
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

@Component({
  template: `
    <div class="context-card-group">
      <app-config-context-card [configRef]="ref" [configObject]="object">
      </app-config-context-card>
      <app-config-context-card [configRef]="ref" [configObject]="object">
      </app-config-context-card>
    </div>
  `,
  imports: [ConfigContextCardComponent],
  standalone: true,
})
class ConfigContextCardGroupsHostComponent {
  readonly ref = new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'});
  readonly object = new ConfigObject({
    kind: Kind.CRAWLJOB,
    id: 'job-1',
    meta: new Meta({name: 'Crawl job'}),
  });
}

describe('ConfigContextCardComponent', () => {
  let fixture: ComponentFixture<ConfigContextCardComponent>;

  const findFact = (label: string): HTMLElement | undefined =>
    [...fixture.nativeElement.querySelectorAll('.context-facts > div')]
      .find((fact: HTMLElement) => fact.querySelector('dt')?.textContent.trim() === label) as HTMLElement | undefined;

  beforeEach(async () => {
    await TestBed.configureTestingModule({
      imports: [
        ConfigContextCardComponent,
        ConfigContextCardHostComponent,
        ConfigContextCardGroupsHostComponent,
      ],
      providers: [provideMaterialAnimationsDisabled(), provideRouter([])],
    }).compileComponents();

    fixture = TestBed.createComponent(ConfigContextCardComponent);
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}));
  });

  it('renders a compact summary with the title as its detail link', async () => {
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

    expect(fixture.nativeElement.querySelector('.context-heading h4').textContent.trim()).toBe('Weekdays');
    const avatar = fixture.nativeElement.querySelector('.context-kind-icon') as HTMLElement;
    expect(avatar).not.toBeNull();
    expect(avatar.querySelector('mat-icon').textContent.trim()).toBe('schedule');
    expect(avatar.querySelector('mat-icon').getAttribute('aria-hidden')).toBe('true');
    expect(fixture.nativeElement.querySelector('.context-item').textContent)
      .not.toContain('Weekday morning schedule');
    const itemStyle = getComputedStyle(fixture.nativeElement.querySelector('.context-item'));
    expect(parseFloat(itemStyle.paddingTop)).toBe(0);
    expect(parseFloat(itemStyle.paddingBottom)).toBe(0);
    expect(fixture.nativeElement.textContent).toContain('0 8 * * 1-5');
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelector('.context-detail-link').getAttribute('href'))
      .toBe('/config/schedule/schedule-1');
    expect(findFact('Valid from')).toBeUndefined();
    expect(findFact('Valid to')).toBeUndefined();
  });

  it('centers facts in a wrapping horizontal row', () => {
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'collection-1',
      kind: Kind.COLLECTION,
    }));
    fixture.detectChanges();

    const factsStyle = getComputedStyle(fixture.nativeElement.querySelector('.context-facts'));
    expect(factsStyle.display).toBe('flex');
    expect(factsStyle.justifyContent).toBe('center');
    expect(factsStyle.flexWrap).toBe('wrap');
    expect(factsStyle.textAlign).toBe('center');
  });

  it('hides an empty fact list when every crawl-job limit is zero', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob(),
    }));
    fixture.detectChanges();

    const facts = fixture.nativeElement.querySelector('.context-facts') as HTMLElement;
    expect(facts.querySelectorAll(':scope > div').length).toBe(0);
    expect(getComputedStyle(facts).display).toBe('none');
  });

  it('shows nonzero numeric facts and hides zero numeric facts before formatting', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLJOB, id: 'job-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'job-1',
      kind: Kind.CRAWLJOB,
      crawlJob: new CrawlJob({limits: {maxDurationS: 60, maxBytes: 0}}),
    }));
    fixture.detectChanges();

    expect(findFact('Maximum duration')?.querySelector('dd')?.textContent.trim()).toBe('1min');
    expect(findFact('Maximum size')).toBeUndefined();
  });

  it('identifies an unavailable related configuration without hiding it', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'missing-id'}));
    fixture.componentRef.setInput('unavailable', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.context-heading h4').textContent.trim()).toBe('missing-id');
    expect(fixture.nativeElement.querySelector('.context-kind-icon mat-icon').textContent.trim())
      .toBe('settings_system_daydream');
    expect(fixture.nativeElement.querySelector('[role="status"]').textContent).toContain('missing-id');
    expect(fixture.nativeElement.querySelector('a')).toBeNull();
  });

  it('leaves outgoing references to projected relationship links', async () => {
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

    expect(fixture.nativeElement.querySelector('.context-facts').textContent)
      .not.toContain('collection-1');
    expect(fixture.nativeElement.querySelector('.context-facts').textContent)
      .not.toContain('browser-config-1');
    expect(fixture.nativeElement.querySelector('.context-facts').textContent)
      .not.toContain('politeness-1');
    expect(fixture.nativeElement.querySelector('.context-footer')).toBeNull();
  });

  it('shows every browser script URL expression without exposing script source', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'script-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
      browserScript: new BrowserScript({
        script: 'window.secretScriptSource = true;',
        urlRegexpList: ['^https://example\\.org/news', '/archive/\\d{4}'],
      }),
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).not.toContain('Show script');
    expect(fixture.nativeElement.textContent).not.toContain('secretScriptSource');
    expect([...fixture.nativeElement.querySelectorAll('.regexp-list code')]
      .map((code: HTMLElement) => code.textContent)).toEqual([
        '^https://example\\.org/news', '/archive/\\d{4}',
      ]);
    expect(fixture.nativeElement.querySelector('details')).toBeNull();
  });

  it('hides empty browser-script URL expressions while retaining its enum default', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'script-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
      browserScript: new BrowserScript({urlRegexpList: []}),
    }));
    fixture.detectChanges();

    expect(findFact('Script type')?.querySelector('dd')?.textContent.trim()).toBe('UNDEFINED');
    expect(findFact('URL regular expressions')).toBeUndefined();
    expect(fixture.nativeElement.querySelector('.regexp-list')).toBeNull();
  });

  it('hides an all-zero viewport but preserves a partially configured viewport', () => {
    const browserConfig = new ConfigObject({id: 'browser-1', kind: Kind.BROWSERCONFIG});
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.BROWSERCONFIG, id: 'browser-1'}));
    fixture.componentRef.setInput('configObject', browserConfig);
    fixture.detectChanges();

    expect(findFact('User agent')).toBeUndefined();
    expect(findFact('Viewport')).toBeUndefined();
    expect(findFact('Page-load timeout')).toBeUndefined();
    expect(findFact('Maximum inactivity')).toBeUndefined();

    browserConfig.browserConfig.windowWidth = 1920;
    fixture.componentRef.setInput('configObject', new ConfigObject(browserConfig));
    fixture.detectChanges();

    expect(findFact('Viewport')?.querySelector('dd')?.textContent.trim()).toBe('1920 × 0');
  });

  it('retains boolean and enum defaults while hiding other zero facts', () => {
    const crawlConfig = new ConfigObject({id: 'crawl-config-1', kind: Kind.CRAWLCONFIG});
    crawlConfig.crawlConfig.extra.createScreenshot = false;
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'crawl-config-1'}));
    fixture.componentRef.setInput('configObject', crawlConfig);
    fixture.detectChanges();

    expect(findFact('Minimum DNS TTL')).toBeUndefined();
    expect(findFact('Priority weight')).toBeUndefined();
    expect(findFact('Screenshots')?.querySelector('dd')?.textContent.trim()).toBe('Disabled');

    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.POLITENESSCONFIG, id: 'politeness-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'politeness-1',
      kind: Kind.POLITENESSCONFIG,
    }));
    fixture.detectChanges();

    expect(findFact('Robots policy')?.querySelector('dd')?.textContent.trim()).toBe('OBEY_ROBOTS');
    expect(findFact('Minimum validity')).toBeUndefined();
    expect(findFact('Hostname mode')?.querySelector('dd')?.textContent.trim()).toBe('No');

    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.COLLECTION, id: 'collection-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'collection-1',
      kind: Kind.COLLECTION,
    }));
    fixture.detectChanges();

    expect(findFact('Deduplication')?.querySelector('dd')?.textContent.trim()).toBe('NONE');
    expect(findFact('Rotation')?.querySelector('dd')?.textContent.trim()).toBe('NONE');
    expect(findFact('File size')).toBeUndefined();
    expect(findFact('Compression')?.querySelector('dd')?.textContent.trim()).toBe('Disabled');
  });

  it('projects relationships after facts and actions', () => {
    const hostFixture = TestBed.createComponent(ConfigContextCardHostComponent);
    hostFixture.detectChanges();

    const item = hostFixture.nativeElement.querySelector('.context-item') as HTMLElement;
    const title = item.querySelector('.context-heading h4');
    const helper = item.querySelector('.context-relations .projected-helper');
    const menu = item.querySelector('.context-menu .projected-menu');
    expect(helper).not.toBeNull();
    expect(menu).not.toBeNull();
    expect(title.compareDocumentPosition(helper) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it('does not divide adjacent cards in the same related-configuration group', () => {
    const hostFixture = TestBed.createComponent(ConfigContextCardGroupsHostComponent);
    hostFixture.detectChanges();

    const groupedCards = hostFixture.nativeElement
      .querySelectorAll('.context-card-group .context-item') as NodeListOf<HTMLElement>;

    expect(groupedCards.length).toBe(2);
    expect(groupedCards[0].classList).not.toContain('context-item-divider');
    expect(groupedCards[1].classList).not.toContain('context-item-divider');
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

    const card = fixture.nativeElement.querySelector('.context-item') as HTMLElement;
    const identity = card.querySelector('.context-identity') as HTMLElement;
    const state = card.querySelector('.deactivated-status') as HTMLElement;
    expect(identity.classList).toContain('inactive');
    expect(card.classList).not.toContain('inactive');
    expect(state.textContent.trim()).toContain('Deactivated');
    expect(state.textContent).not.toContain('Crawljob status:');
    expect(card.querySelector('.status-badge')).toBeNull();
  });

});
