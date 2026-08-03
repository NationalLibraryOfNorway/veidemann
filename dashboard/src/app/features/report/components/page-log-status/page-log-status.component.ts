import {ChangeDetectionStrategy, Component, inject, Input, OnChanges} from '@angular/core';
import {MAT_DIALOG_DATA, MatDialog, MatDialogModule} from '@angular/material/dialog';
import {MatButtonModule} from '@angular/material/button';
import {MatCardModule} from '@angular/material/card';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatListModule} from '@angular/material/list';
import {MatTabsModule} from '@angular/material/tabs';
import {MatTooltipModule} from '@angular/material/tooltip';
import {RouterLink} from '@angular/router';

import {PageLog, Resource} from '../../../../shared/models';

interface OutlinkView {
  raw: string;
  href: string | null;
  host: string;
  path: string;
}

interface ResourceMetadata {
  label: string;
  value: string;
}

@Component({
  selector: 'app-resource-metadata-dialog',
  template: `
    <h2 mat-dialog-title>Resource metadata</h2>
    <mat-dialog-content>
      <dl class="metadata-list">
        @for (item of data; track item.label) {
          <div><dt>{{item.label}}</dt><dd>{{item.value}}</dd></div>
        }
      </dl>
    </mat-dialog-content>
    <mat-dialog-actions align="end"><button mat-button mat-dialog-close>Close</button></mat-dialog-actions>
  `,
  styles: [`
    .metadata-list { display: grid; gap: 12px; margin: 0; }
    .metadata-list div { display: grid; grid-template-columns: minmax(8rem, 1fr) 2fr; gap: 16px; }
    dt { color: var(--mat-sys-on-surface-variant); }
    dd { margin: 0; overflow-wrap: anywhere; }
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
  styleUrls: ['./page-log-status.component.css'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  standalone: true,
  imports: [
    MatButtonModule,
    MatCardModule,
    MatDialogModule,
    MatFormFieldModule,
    MatIconModule,
    MatInputModule,
    MatListModule,
    MatTabsModule,
    MatTooltipModule,
    RouterLink,
  ]
})
export class PageLogStatusComponent implements OnChanges {
  private readonly dialog = inject(MatDialog);
  @Input() pageLog: PageLog;

  filteredResources: readonly Resource[] = [];
  filteredOutlinks: readonly OutlinkView[] = [];
  private resourceFilter = '';
  private outlinkFilter = '';
  private outlinks: readonly OutlinkView[] = [];

  ngOnChanges(): void {
    this.outlinks = (this.pageLog?.outlink ?? []).map(value => this.parseOutlink(value));
    this.applyResourceFilter(this.resourceFilter);
    this.applyOutlinkFilter(this.outlinkFilter);
  }

  applyResourceFilter(value: string): void {
    this.resourceFilter = value.trim().toLocaleLowerCase();
    this.filteredResources = (this.pageLog?.resource ?? []).filter(resource => !this.resourceFilter || [
      resource.uri,
      resource.mimeType,
      resource.resourceType,
      resource.discoveryPath,
      resource.statusCode,
      resource.error?.msg,
      resource.error?.detail,
    ].some(candidate => String(candidate ?? '').toLocaleLowerCase().includes(this.resourceFilter)));
  }

  applyOutlinkFilter(value: string): void {
    this.outlinkFilter = value.trim().toLocaleLowerCase();
    this.filteredOutlinks = this.outlinks.filter(outlink =>
      !this.outlinkFilter || outlink.raw.toLocaleLowerCase().includes(this.outlinkFilter));
  }

  hasError(resource: Resource): boolean {
    return !!(resource.error?.code || resource.error?.msg || resource.error?.detail);
  }

  showMetadata(resource: Resource): void {
    const metadata: ResourceMetadata[] = [
      ['URI', resource.uri],
      ['Status code', resource.statusCode],
      ['MIME type', resource.mimeType],
      ['Resource type', resource.resourceType],
      ['Discovery path', resource.discoveryPath],
      ['From cache', resource.fromCache],
      ['Renderable', resource.renderable],
      ['WARC ID', resource.warcId],
      ['Referrer', resource.referrer],
      ['Method', resource.method],
      ['Error', this.hasError(resource)
        ? [resource.error.code, resource.error.msg, resource.error.detail].filter(Boolean).join(': ')
        : ''],
    ].filter(([, value]) => value !== '' && value !== null && value !== undefined)
      .map(([label, value]) => ({label: String(label), value: String(value)}));
    this.dialog.open(ResourceMetadataDialogComponent, {data: metadata, autoFocus: 'dialog'});
  }

  private parseOutlink(raw: string): OutlinkView {
    try {
      const url = new URL(raw);
      if (url.protocol !== 'http:' && url.protocol !== 'https:') {
        throw new Error('Unsupported URL scheme');
      }
      return {
        raw,
        href: url.href,
        host: url.host,
        path: `${url.pathname}${url.search}${url.hash}` || '/',
      };
    } catch {
      return {raw, href: null, host: 'Invalid URI', path: raw};
    }
  }
}
