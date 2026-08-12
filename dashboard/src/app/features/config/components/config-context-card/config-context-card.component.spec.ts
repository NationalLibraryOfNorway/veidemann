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

  it('shows Not set when a browser script has no URL expressions', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.BROWSERSCRIPT, id: 'script-1'}));
    fixture.componentRef.setInput('configObject', new ConfigObject({
      id: 'script-1',
      kind: Kind.BROWSERSCRIPT,
      browserScript: new BrowserScript({urlRegexpList: []}),
    }));
    fixture.detectChanges();

    const regexpFact = [...fixture.nativeElement.querySelectorAll('.context-facts > div')]
      .find((fact: HTMLElement) => fact.querySelector('dt')?.textContent.trim() === 'URL regular expressions');
    expect(regexpFact?.querySelector('dd')?.textContent.trim()).toBe('Not set');
    expect(fixture.nativeElement.querySelector('.regexp-list')).toBeNull();
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
