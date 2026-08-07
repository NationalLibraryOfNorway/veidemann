import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {provideRouter} from '@angular/router';

import {provideMaterialAnimationsDisabled} from '../../../../core/core.testing.module';
import {
  BrowserScript,
  ConfigObject,
  ConfigRef,
  CrawlConfig,
  CrawlScheduleConfig,
  Kind,
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
    expect(fixture.nativeElement.querySelector('mat-card-subtitle').textContent.trim())
      .toBe('Weekday morning schedule');
    expect(fixture.nativeElement.querySelector('[mat-card-avatar]')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-card-content').textContent)
      .not.toContain('Weekday morning schedule');
    expect(fixture.nativeElement.textContent).toContain('0 8 * * 1-5');
    expect(fixture.nativeElement.querySelector('mat-card').getAttribute('appearance')).toBe('outlined');
    expect(fixture.nativeElement.querySelector('a').getAttribute('href')).toContain('/config/schedule/schedule-1');
  });

  it('identifies an unavailable related configuration without hiding it', () => {
    fixture.componentRef.setInput('configRef', new ConfigRef({kind: Kind.CRAWLCONFIG, id: 'missing-id'}));
    fixture.componentRef.setInput('unavailable', true);
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('[role="status"]').textContent).toContain('missing-id');
    expect(fixture.nativeElement.querySelector('a')).toBeNull();
  });

  it('links configuration IDs to their detail pages', async () => {
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

    const hrefs = Array.from(
      fixture.nativeElement.querySelectorAll('.summary-grid a.mat-mdc-button') as NodeListOf<HTMLAnchorElement>
    ).map(link => link.getAttribute('href'));
    expect(hrefs).toEqual([
      '/config/crawlconfig/crawl-config-1',
      '/config/collection/collection-1',
      '/config/browserconfig/browser-config-1',
      '/config/politenessconfig/politeness-1',
    ]);
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

  it('projects helper content into the card header after its subtitle', () => {
    const hostFixture = TestBed.createComponent(ConfigContextCardHostComponent);
    hostFixture.detectChanges();

    const header = hostFixture.nativeElement.querySelector('mat-card-header') as HTMLElement;
    const subtitle = header.querySelector('mat-card-subtitle');
    const helper = header.querySelector('.projected-helper');
    expect(helper).not.toBeNull();
    expect(subtitle.compareDocumentPosition(helper) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });
});
