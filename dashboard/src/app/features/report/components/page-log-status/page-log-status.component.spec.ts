import {ComponentFixture, TestBed} from '@angular/core/testing';
import {Component} from '@angular/core';
import {PageLogStatusComponent} from './page-log-status.component';
import {provideCoreTesting} from '../../../../core/core.testing.module';
import {PageLog, Resource} from '../../../../shared/models';
import {provideRouter, Router} from '@angular/router';
import {MatSort} from '@angular/material/sort';

@Component({
  template: `
    <app-page-log-status [pageLog]="pageLog">
      <span detailHeaderHelpers class="projected-helper">Helper</span>
    </app-page-log-status>
  `,
  imports: [PageLogStatusComponent],
  standalone: true,
})
class PageLogStatusHostComponent {
  pageLog = new PageLog();
}

describe('PageLogStatusComponent', () => {
  let component: PageLogStatusComponent;
  let fixture: ComponentFixture<PageLogStatusComponent>;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [PageLogStatusComponent, PageLogStatusHostComponent],
      providers: [...provideCoreTesting, provideRouter([])]
    })
      .compileComponents();
  });

  beforeEach(async () => {
    fixture = TestBed.createComponent(PageLogStatusComponent);
    component = fixture.componentInstance;
    await fixture.whenStable();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('shows tab counts, filters resources, and preserves malformed outlinks', () => {
    const pageLog = new PageLog({
      resource: [
        new Resource({method: 'GET', uri: 'https://example.org/image.png', mimeType: 'image/png'}),
        new Resource({method: 'GET', uri: 'https://example.org/app.js', mimeType: 'text/javascript'}),
      ],
      outlink: ['https://example.org/path', 'not a uri'],
    });
    fixture.componentRef.setInput('pageLog', pageLog);
    fixture.detectChanges();

    expect(fixture.nativeElement.textContent).toContain('Resources (2)');
    expect(fixture.nativeElement.textContent).toContain('Outlinks (2)');
    component.applyResourceFilter('IMAGE');
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['https://example.org/image.png']);
    component.applyOutlinkFilter('not a uri');
    expect(component.filteredOutlinks).toEqual([expect.objectContaining({raw: 'not a uri', href: null})]);
  });

  it('offers MIME, resource-type, status-family, and exact-status facets and combines them with search', () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      resource: [
        new Resource({uri: 'ok-image', mimeType: 'Image/PNG', resourceType: 'Image', statusCode: 200}),
        new Resource({uri: 'missing-image', mimeType: 'image/png', resourceType: 'Image', statusCode: 404}),
        new Resource({uri: 'broken-script', mimeType: 'text/javascript; charset=utf-8', resourceType: 'Script', statusCode: 500}),
      ],
    }));
    fixture.detectChanges();

    expect(component.resourceMimeTypes()).toEqual(['image/png', 'text/javascript']);
    expect(component.resourceTypes()).toEqual(['Image', 'Script']);
    expect(component.resourceStatusFamilies()).toEqual([2, 4, 5]);
    expect(component.resourceStatusCodes()).toEqual([200, 404, 500]);

    component.applyResourceExactStatusFilter([200]);
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['ok-image']);
    component.applyResourceExactStatusFilter([]);

    component.applyResourceMimeTypeFilter(['image/png']);
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['ok-image', 'missing-image']);

    component.applyResourceStatusFilter([4]);
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['missing-image']);

    component.applyResourceTypeFilter(['Script']);
    expect(component.filteredResources).toEqual([]);

    component.applyResourceMimeTypeFilter([]);
    component.applyResourceStatusFilter([5]);
    component.applyResourceFilter('broken');
    expect(component.filteredResources.map(resource => resource.uri)).toEqual(['broken-script']);
  });

  it('shows only the general and exact status chips present in loaded resources', () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      resource: [
        new Resource({uri: 'first', statusCode: 200}),
        new Resource({uri: 'second', statusCode: 200}),
      ],
    }));
    fixture.detectChanges();

    const chips = [...fixture.nativeElement.querySelectorAll(
      'app-http-status-filter mat-chip-option'
    )] as HTMLElement[];
    expect(chips.map(chip => chip.textContent?.trim())).toEqual(['2xx', '200']);
  });

  it('uses a card-free full-width tab layout and renders valid outlinks as table links', async () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      uri: 'https://example.org',
      collectionFinalName: 'archive-collection',
      outlink: ['https://outlink.example/path'],
    }));
    fixture.detectChanges();

    expect(fixture.nativeElement.querySelector('.detail-header h1')?.textContent).toBe('https://example.org');
    expect(fixture.nativeElement.querySelector('.collection-metadata dt')?.textContent).toBe('Collection');
    expect(fixture.nativeElement.querySelector('.collection-metadata dd')?.textContent).toBe('archive-collection');
    expect(fixture.nativeElement.querySelector('mat-card')).toBeNull();
    expect(fixture.nativeElement.querySelector('mat-tab-group.detail-tabs')).not.toBeNull();
    const resourceToolbar = fixture.nativeElement.querySelector('.resource-filter-toolbar') as HTMLElement;
    expect(getComputedStyle(resourceToolbar).marginLeft).toBe('16px');
    expect(getComputedStyle(resourceToolbar).marginRight).toBe('16px');
    const tabs = fixture.nativeElement.querySelectorAll('[role="tab"]') as NodeListOf<HTMLElement>;
    tabs[1].click();
    await fixture.whenStable();
    fixture.detectChanges();
    const link = fixture.nativeElement.querySelector('table[aria-label="Page outlinks"] a[target="_blank"]') as HTMLAnchorElement;
    const outlinkContent = link.closest('.tab-content') as HTMLElement;
    const outlinkSearch = outlinkContent.querySelector(':scope > .search-field') as HTMLElement;
    const domainFilter = outlinkContent.querySelector('.outlink-domain-filter') as HTMLElement;
    expect(link.href).toBe('https://outlink.example/path');
    expect(link.closest('.table-scroll')).not.toBeNull();
    expect(getComputedStyle(outlinkSearch).marginInline).toBe('16px');
    expect(getComputedStyle(domainFilter).marginLeft).toBe('16px');
    expect(getComputedStyle(domainFilter).marginRight).toBe('16px');
  });

  it('offers unique domain chips and combines domain and text filters', async () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      outlink: [
        'https://www.example.org/first',
        'https://archive.example.org/page',
        'https://www.example.org/second',
        'not a uri',
      ],
    }));
    fixture.detectChanges();

    const tabs = fixture.nativeElement.querySelectorAll('[role="tab"]') as NodeListOf<HTMLElement>;
    tabs[1].click();
    await fixture.whenStable();
    fixture.detectChanges();

    const chips = [...fixture.nativeElement.querySelectorAll('.outlink-domain-filter mat-chip-option')]
      .map((chip: Element) => chip.textContent?.trim());
    expect(chips).toEqual(['archive.example.org', 'www.example.org']);
    const domainScroll = fixture.nativeElement.querySelector('.outlink-domain-scroll') as HTMLElement;
    expect(getComputedStyle(domainScroll).maxHeight).toBe('160px');
    expect(getComputedStyle(domainScroll).overflowY).toBe('auto');

    component.applyOutlinkDomainFilter(['www.example.org']);
    expect(component.filteredOutlinks.map(outlink => outlink.raw)).toEqual([
      'https://www.example.org/first',
      'https://www.example.org/second',
    ]);

    component.applyOutlinkFilter('second');
    expect(component.filteredOutlinks.map(outlink => outlink.raw))
      .toEqual(['https://www.example.org/second']);

    component.applyOutlinkDomainFilter([]);
    component.applyOutlinkFilter('not a uri');
    expect(component.filteredOutlinks).toEqual([expect.objectContaining({raw: 'not a uri', domain: null})]);
  });

  it('projects shortcuts into the standard page header', () => {
    const hostFixture = TestBed.createComponent(PageLogStatusHostComponent);
    hostFixture.detectChanges();

    expect(hostFixture.nativeElement.querySelector('.detail-header .projected-helper')).not.toBeNull();
  });

  it('opens a metadata dialog with resource error details', () => {
    const dialog = component['dialog'];
    const open = vi.spyOn(dialog, 'open').mockReturnValue({} as never);
    const resource = new Resource({uri: 'https://example.org', statusCode: 500});
    resource.error.code = 7;
    resource.error.msg = 'Failed';

    component.showMetadata(resource);

    expect(open).toHaveBeenCalledWith(expect.anything(), expect.objectContaining({
      data: expect.arrayContaining([expect.objectContaining({label: 'Error', value: '7: Failed'})]),
      width: '42rem',
      maxWidth: '95vw',
    }));
  });

  it('uses a fixed-layout responsive resource table with a complete details action', () => {
    const uri = 'https://example.org/a/very/long/resource/path/image.png?cache=1234567890';
    fixture.componentRef.setInput('pageLog', new PageLog({
      resource: [new Resource({uri, statusCode: 200, mimeType: 'image/png'})],
    }));
    fixture.detectChanges();

    const table = fixture.nativeElement.querySelector('table[aria-label="Page resources"]');
    const uriValue = table.querySelector('a.uri-value') as HTMLAnchorElement;
    const detailsButton = table.querySelector('button[aria-label="Show resource metadata"]');

    expect(table.classList).toContain('resource-table');
    expect(component.resourceColumns[0]).toBe('method');
    expect(table.closest('.resource-table-container')).not.toBeNull();
    expect(uriValue.textContent).toBe(uri);
    expect(uriValue.href).toBe(uri);
    expect(uriValue.target).toBe('_blank');
    expect(uriValue.rel).toBe('noopener noreferrer');
    expect(getComputedStyle(uriValue).display).toBe('inline-block');
    expect(getComputedStyle(uriValue).width).toBe('fit-content');
    expect(getComputedStyle(uriValue).maxWidth).toBe('100%');
    expect(detailsButton).not.toBeNull();
  });

  it('opens the matching crawl-log detail from the resource row without hijacking URI or metadata clicks', () => {
    const uri = 'https://example.org/image.png';
    fixture.componentRef.setInput('pageLog', new PageLog({
      resource: [new Resource({uri, warcId: 'warc-resource-1'})],
    }));
    fixture.detectChanges();
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const showMetadata = vi.spyOn(component, 'showMetadata').mockImplementation(() => undefined);
    const row = fixture.nativeElement.querySelector('.resource-row') as HTMLTableRowElement;
    const uriLink = row.querySelector('a.uri-value') as HTMLAnchorElement;
    const metadataButton = row.querySelector(
      'button[aria-label="Show resource metadata"]'
    ) as HTMLButtonElement;

    expect(row.classList).toContain('resource-row-link');
    expect(row.tabIndex).toBe(0);

    row.click();
    expect(navigate).toHaveBeenLastCalledWith(['/report', 'crawllog', 'warc-resource-1']);

    navigate.mockClear();
    uriLink.addEventListener('click', event => event.preventDefault());
    uriLink.click();
    expect(navigate).not.toHaveBeenCalled();

    metadataButton.click();
    expect(navigate).not.toHaveBeenCalled();
    expect(showMetadata).toHaveBeenCalledOnce();

    row.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(navigate).toHaveBeenCalledWith(['/report', 'crawllog', 'warc-resource-1']);
  });

  it('does not make resource rows without a WARC ID navigable', () => {
    fixture.componentRef.setInput('pageLog', new PageLog({
      resource: [new Resource({uri: 'https://example.org/no-warc'})],
    }));
    fixture.detectChanges();
    const navigate = vi.spyOn(TestBed.inject(Router), 'navigate').mockResolvedValue(true);
    const row = fixture.nativeElement.querySelector('.resource-row') as HTMLTableRowElement;

    expect(row.classList).not.toContain('resource-row-link');
    expect(row.getAttribute('tabindex')).toBeNull();
    row.click();
    row.dispatchEvent(new KeyboardEvent('keydown', {key: 'Enter', bubbles: true}));
    expect(navigate).not.toHaveBeenCalled();
  });

  it('sorts resource status codes numerically and outlinks alphabetically', () => {
    const resources = [
      new Resource({uri: 'third', statusCode: 404}),
      new Resource({uri: 'first', statusCode: 20}),
      new Resource({uri: 'second', statusCode: 100}),
    ];
    const sortedResources = component.resources.sortData(resources, {
      active: 'statusCode', direction: 'asc',
    } as MatSort);
    expect(sortedResources.map(resource => resource.statusCode)).toEqual([20, 100, 404]);

    fixture.componentRef.setInput('pageLog', new PageLog({outlink: ['https://z.test', 'https://a.test']}));
    fixture.detectChanges();
    const sortedOutlinks = component.outlinks.sortData(component.outlinks.data, {
      active: 'uri', direction: 'asc',
    } as MatSort);
    expect(sortedOutlinks.map(outlink => outlink.raw)).toEqual(['https://a.test', 'https://z.test']);
  });
});
