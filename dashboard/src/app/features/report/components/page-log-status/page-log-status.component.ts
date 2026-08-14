import {
  ChangeDetectionStrategy,
  Component,
  ErrorHandler,
  inject,
  Input,
  OnChanges,
  signal,
  ViewChild,
} from '@angular/core';
import {MatButtonModule} from '@angular/material/button';
import {MatChipsModule} from '@angular/material/chips';
import {MAT_DIALOG_DATA, MatDialog, MatDialogModule} from '@angular/material/dialog';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatSort, MatSortModule} from '@angular/material/sort';
import {MatTableDataSource, MatTableModule} from '@angular/material/table';
import {MatTabsModule} from '@angular/material/tabs';
import {MatTooltipModule} from '@angular/material/tooltip';
import {Router} from '@angular/router';

import {PageLog, Resource} from '../../../../shared/models';
import {DetailHeaderComponent} from '../../../../shared/components';
import {
  HttpStatusFamily,
  HttpStatusFilterComponent,
  httpStatusFamily,
  uniqueHttpStatusCodes,
} from '../http-status-filter/http-status-filter.component';

interface OutlinkView {
  raw: string;
  href: string | null;
  domain: string | null;
}

interface OutlinkFilter {
  search: string;
  domains: string[];
}

interface ResourceFilter {
  search: string;
  mimeTypes: string[];
  resourceTypes: string[];
  statusFamilies: HttpStatusFamily[];
  statusCodes: number[];
}

interface ResourceMetadata {
  label: string;
  value: string;
}

@Component({
  selector: 'app-resource-metadata-dialog',
  template: `
    <h2 mat-dialog-title i18n="@@resourceMetadataDialogTitle">Resource metadata</h2>
    <mat-dialog-content>
      <dl class="metadata-list">
        @for (item of data; track item.label) {
          <div><dt>{{item.label}}</dt><dd>{{item.value}}</dd></div>
        }
      </dl>
    </mat-dialog-content>
    <mat-dialog-actions align="end"><button mat-button mat-dialog-close i18n="@@commonButtonClose">CLOSE</button></mat-dialog-actions>
  `,
  styles: [`
    .metadata-list { display: grid; gap: 12px; margin: 0; }
    .metadata-list div { display: grid; gap: 4px; }
    dt { color: var(--mat-sys-on-surface-variant); }
    dd { margin: 0; overflow-wrap: anywhere; }
    @media (min-width: 600px) {
      .metadata-list div { grid-template-columns: minmax(8rem, 1fr) 2fr; gap: 16px; }
    }
  `],
  imports: [MatButtonModule, MatDialogModule],
  standalone: true,
})
export class ResourceMetadataDialogComponent {
  readonly data = inject<ResourceMetadata[]>(MAT_DIALOG_DATA);
}

@Component({
  selector: 'app-page-log-status',
  templateUrl: './page-log-status.component.html',
  styleUrls: ['../detail-status-layout.scss', './page-log-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    MatButtonModule,
    MatChipsModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatSortModule,
    MatTableModule,
    MatTabsModule,
    MatTooltipModule,
    DetailHeaderComponent,
    HttpStatusFilterComponent,
  ]
})
export class PageLogStatusComponent implements OnChanges {
  private readonly dialog = inject(MatDialog);
  private readonly errorHandler = inject(ErrorHandler);
  private readonly router = inject(Router);
  @Input() pageLog: PageLog;

  readonly resourceColumns = [
    'method', 'uri', 'statusCode', 'mimeType', 'resourceType', 'discoveryPath', 'error', 'metadata',
  ];
  readonly outlinkColumns = ['uri'];
  readonly resources = new MatTableDataSource<Resource>();
  readonly outlinks = new MatTableDataSource<OutlinkView>();
  readonly outlinkDomains = signal<readonly string[]>([]);
  readonly selectedOutlinkDomains = signal<string[]>([]);
  readonly outlinkSearch = signal('');
  readonly resourceMimeTypes = signal<readonly string[]>([]);
  readonly resourceTypes = signal<readonly string[]>([]);
  readonly resourceStatusFamilies = signal<readonly HttpStatusFamily[]>([]);
  readonly resourceStatusCodes = signal<readonly number[]>([]);
  readonly selectedResourceMimeTypes = signal<string[]>([]);
  readonly selectedResourceTypes = signal<string[]>([]);
  readonly selectedResourceStatusFamilies = signal<HttpStatusFamily[]>([]);
  readonly selectedResourceStatusCodes = signal<number[]>([]);
  readonly resourceSearch = signal('');

  @ViewChild('resourceSort') set resourceSort(sort: MatSort | undefined) {
    if (sort) this.resources.sort = sort;
  }
  @ViewChild('outlinkSort') set outlinkSort(sort: MatSort | undefined) {
    if (sort) this.outlinks.sort = sort;
  }

  constructor() {
    this.resources.filterPredicate = (resource, filterValue) => {
      const filter = JSON.parse(filterValue) as ResourceFilter;
      const matchesSearch = [
        resource.method,
        resource.uri,
        resource.mimeType,
        resource.resourceType,
        resource.discoveryPath,
        resource.statusCode,
        resource.error?.code,
        resource.error?.msg,
        resource.error?.detail,
      ].some(candidate => String(candidate ?? '').toLocaleLowerCase().includes(filter.search));
      return matchesSearch &&
        (filter.mimeTypes.length === 0 || filter.mimeTypes.includes(normalizeMimeType(resource.mimeType))) &&
        (filter.resourceTypes.length === 0 || filter.resourceTypes
          .some(resourceType => normalizeResourceType(resourceType) === normalizeResourceType(resource.resourceType))) &&
        (filter.statusFamilies.length === 0 && filter.statusCodes.length === 0 ||
          filter.statusFamilies.includes(httpStatusFamily(resource.statusCode) as HttpStatusFamily) ||
          filter.statusCodes.includes(resource.statusCode));
    };
    this.resources.sortingDataAccessor = (resource, column) =>
      column === 'statusCode' ? Number(resource.statusCode ?? 0) :
        String(resource[column as keyof Resource] ?? '').toLocaleLowerCase();
    this.outlinks.filterPredicate = (outlink, filter) => {
      const {search, domains} = JSON.parse(filter) as OutlinkFilter;
      return outlink.raw.toLocaleLowerCase().includes(search) &&
        (domains.length === 0 || (!!outlink.domain && domains.includes(outlink.domain)));
    };
    this.outlinks.sortingDataAccessor = outlink => outlink.raw.toLocaleLowerCase();
  }

  ngOnChanges(): void {
    const resources = this.pageLog?.resource ?? [];
    this.resources.data = resources;
    this.resourceMimeTypes.set(uniqueFacetValues(resources.map(resource => resource.mimeType), normalizeMimeType));
    this.resourceTypes.set(uniqueResourceTypes(resources.map(resource => resource.resourceType)));
    const statusCodes = uniqueHttpStatusCodes(resources.map(resource => resource.statusCode));
    this.resourceStatusCodes.set(statusCodes);
    this.resourceStatusFamilies.set([...new Set(statusCodes
      .map(httpStatusFamily)
      .filter((family): family is HttpStatusFamily => family !== null))]);
    this.selectedResourceMimeTypes.set([]);
    this.selectedResourceTypes.set([]);
    this.selectedResourceStatusFamilies.set([]);
    this.selectedResourceStatusCodes.set([]);
    this.resourceSearch.set('');
    this.updateResourceFilter();
    const outlinks = (this.pageLog?.outlink ?? []).map(value => this.parseOutlink(value));
    this.outlinks.data = outlinks;
    this.outlinkDomains.set([...new Set(outlinks
      .map(outlink => outlink.domain)
      .filter((domain): domain is string => !!domain))]
      .sort((left, right) => left.localeCompare(right)));
    this.selectedOutlinkDomains.set([]);
    this.outlinkSearch.set('');
    this.updateOutlinkFilter();
  }

  applyResourceFilter(value: string): void {
    this.resourceSearch.set(value.trim().toLocaleLowerCase());
    this.updateResourceFilter();
  }

  applyResourceMimeTypeFilter(mimeTypes: string[] | null): void {
    this.selectedResourceMimeTypes.set(mimeTypes ?? []);
    this.updateResourceFilter();
  }

  applyResourceTypeFilter(resourceTypes: string[] | null): void {
    this.selectedResourceTypes.set(resourceTypes ?? []);
    this.updateResourceFilter();
  }

  applyResourceStatusFilter(statusFamilies: HttpStatusFamily[]): void {
    this.selectedResourceStatusFamilies.set(statusFamilies);
    this.updateResourceFilter();
  }

  applyResourceExactStatusFilter(statusCodes: number[]): void {
    this.selectedResourceStatusCodes.set(statusCodes);
    this.updateResourceFilter();
  }

  applyOutlinkFilter(value: string): void {
    this.outlinkSearch.set(value.trim().toLocaleLowerCase());
    this.updateOutlinkFilter();
  }

  applyOutlinkDomainFilter(domains: string[] | null): void {
    this.selectedOutlinkDomains.set(domains ?? []);
    this.updateOutlinkFilter();
  }

  resourceTabLabel(count: number): string {
    return $localize`:@@pageResourcesTabLabel:Resources (${count}:RESOURCE_COUNT:)`;
  }

  outlinkTabLabel(count: number): string {
    return $localize`:@@pageOutlinksTabLabel:Outlinks (${count}:OUTLINK_COUNT:)`;
  }

  get filteredResources(): readonly Resource[] { return this.resources.filteredData; }
  get filteredOutlinks(): readonly OutlinkView[] { return this.outlinks.filteredData; }

  hasError(resource: Resource): boolean {
    return !!(resource.error?.code || resource.error?.msg || resource.error?.detail);
  }

  showMetadata(resource: Resource): void {
    const metadata: ResourceMetadata[] = [
      this.metadata($localize`:@@resourceMetadataUri:URI`, resource.uri),
      this.metadata($localize`:@@resourceMetadataFromCache:From cache`, this.booleanLabel(resource.fromCache)),
      this.metadata($localize`:@@resourceMetadataRenderable:Renderable`, this.booleanLabel(resource.renderable)),
      this.metadata($localize`:@@resourceMetadataResourceType:Resource type`, resource.resourceType),
      this.metadata($localize`:@@resourceMetadataMimeType:MIME type`, resource.mimeType),
      this.metadata($localize`:@@resourceMetadataStatusCode:Status code`, resource.statusCode || null),
      this.metadata($localize`:@@resourceMetadataDiscoveryPath:Discovery path`, resource.discoveryPath),
      this.metadata($localize`:@@resourceMetadataWarcId:WARC ID`, resource.warcId),
      this.metadata($localize`:@@resourceMetadataReferrer:Referrer`, resource.referrer),
      this.metadata($localize`:@@resourceMetadataErrorCode:Error code`, resource.error?.code || null),
      this.metadata($localize`:@@resourceMetadataErrorMessage:Error message`, resource.error?.msg),
      this.metadata($localize`:@@resourceMetadataErrorDetails:Error details`, resource.error?.detail),
      this.metadata($localize`:@@resourceMetadataMethod:Method`, resource.method),
    ];
    this.dialog.open(ResourceMetadataDialogComponent, {
      data: metadata,
      autoFocus: 'dialog',
      width: '42rem',
      maxWidth: '95vw',
    });
  }

  private metadata(label: string, value: string | number | null | undefined): ResourceMetadata {
    return {
      label,
      value: value === '' || value === null || value === undefined
        ? $localize`:@@commonNotAvailable:Not available`
        : String(value),
    };
  }

  private booleanLabel(value: boolean): string {
    return value ? $localize`:@@commonYes:Yes` : $localize`:@@commonNo:No`;
  }

  onResourceRowClick(resource: Resource, event: Event): void {
    if (!resource.warcId || this.isInteractiveTarget(event.target)) {
      return;
    }
    this.openCrawlLog(resource);
  }

  onResourceRowKeydown(resource: Resource, event: KeyboardEvent): void {
    if (event.key !== 'Enter' || !resource.warcId || this.isInteractiveTarget(event.target)) {
      return;
    }
    event.preventDefault();
    this.openCrawlLog(resource);
  }

  private parseOutlink(raw: string): OutlinkView {
    try {
      const url = new URL(raw);
      if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error('Unsupported URL scheme');
      return {raw, href: url.href, domain: url.hostname};
    } catch {
      return {raw, href: null, domain: null};
    }
  }

  private isInteractiveTarget(target: EventTarget | null): boolean {
    return target instanceof Element && !!target.closest('a, button, input, [role="button"]');
  }

  private openCrawlLog(resource: Resource): void {
    this.router.navigate(['/report', 'crawllog', resource.warcId])
      .catch(error => this.errorHandler.handleError(error));
  }

  private updateOutlinkFilter(): void {
    this.outlinks.filter = JSON.stringify({
      search: this.outlinkSearch(),
      domains: this.selectedOutlinkDomains(),
    } satisfies OutlinkFilter);
  }

  private updateResourceFilter(): void {
    this.resources.filter = JSON.stringify({
      search: this.resourceSearch(),
      mimeTypes: this.selectedResourceMimeTypes(),
      resourceTypes: this.selectedResourceTypes(),
      statusFamilies: this.selectedResourceStatusFamilies(),
      statusCodes: this.selectedResourceStatusCodes(),
    } satisfies ResourceFilter);
  }
}

function normalizeMimeType(mimeType: string): string {
  return mimeType.split(';', 1)[0].trim().toLocaleLowerCase();
}

function normalizeResourceType(resourceType: string): string {
  return resourceType.trim().toLocaleLowerCase();
}

function uniqueFacetValues(values: readonly string[], normalize: (value: string) => string): string[] {
  return [...new Set(values.map(normalize).filter(value => value.length > 0))]
    .sort((left, right) => left.localeCompare(right));
}

function uniqueResourceTypes(values: readonly string[]): string[] {
  const resourceTypes = new Map<string, string>();
  values.forEach(value => {
    const displayValue = value.trim();
    if (displayValue) resourceTypes.set(normalizeResourceType(displayValue), displayValue);
  });
  return [...resourceTypes.values()].sort((left, right) => left.localeCompare(right));
}
