import { AfterViewInit,ChangeDetectionStrategy,ChangeDetectorRef,Component,ElementRef,inject,Input,OnChanges,ViewChild } from '@angular/core';
import { FormsModule,ReactiveFormsModule } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatDialog } from '@angular/material/dialog';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInput } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { MatTooltipModule } from '@angular/material/tooltip';
import { filter,take } from 'rxjs/operators';
import { QueryComponent } from '../../../../shared/components/query.component';
import { ConfigQuery } from '../../../../shared/func';
import { BrowserScriptType, Kind, Role, RobotsPolicy } from '../../../../shared/models';
import {parseConfigSearchTerm, serializeConfigSearchTerm} from '../../func/query';
import type {ConfigLabelSelector} from '../../func/query';
import type {ConfigOptions} from '../../func/options';

interface SelectedCrawlJob {
  id: string;
  name: string;
}

@Component({
  selector: 'app-config-query',
  styleUrls: ['config-query.component.scss'],
  templateUrl: './config-query.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FormsModule,
    MatButtonModule,
    MatChipsModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatSelectModule,
    MatTooltipModule,
    ReactiveFormsModule
  ],
  standalone: true
})
export class ConfigQueryComponent extends QueryComponent<ConfigQuery> implements OnChanges, AfterViewInit {

  private readonly dialog = inject(MatDialog);
  private readonly cdr = inject(ChangeDetectorRef);

  readonly Kind = Kind;
  readonly BrowserScriptType = BrowserScriptType;
  readonly RobotsPolicy = RobotsPolicy;
  readonly Role = Role;

  term: string;
  appliedLabelSearch: ConfigLabelSelector | null = null;

  @Input()
  options: ConfigOptions;

  get filterBrowserScriptTypes(): BrowserScriptType[] {
    return (this.options?.browserScriptTypes ?? [])
      .filter(type => type !== BrowserScriptType.UNDEFINED);
  }

  @ViewChild('search') searchElement: ElementRef;

  get selectedCrawlJobs(): SelectedCrawlJob[] {
    const selectedIds = this.form.controls['crawlJobIdList'].value as string[] | null;
    return (selectedIds ?? []).map(id => ({
      id,
      name: this.options?.crawlJobs?.find(crawlJob => crawlJob.id === id)?.meta?.name || id,
    }));
  }

  get showLabelSearchHelpers(): boolean {
    return !!this.query?.kind
      && this.query.kind !== Kind.ROLEMAPPING
      && !this.term
      && !this.appliedLabelSearch;
  }

  get searchLabel(): string {
    switch (this.query?.kind) {
      case Kind.CRAWLENTITY:
        return $localize`:@@configQuerySearchEntitiesLabel:Search entities`;
      case Kind.SEED:
        return $localize`:@@configQuerySearchSeedsLabel:Search seeds`;
      case Kind.CRAWLJOB:
        return $localize`:@@configQuerySearchCrawlJobsLabel:Search crawl jobs`;
      case Kind.CRAWLSCHEDULECONFIG:
        return $localize`:@@configQuerySearchSchedulesLabel:Search schedules`;
      case Kind.CRAWLCONFIG:
        return $localize`:@@configQuerySearchCrawlConfigsLabel:Search crawl configs`;
      case Kind.COLLECTION:
        return $localize`:@@configQuerySearchCollectionsLabel:Search collections`;
      case Kind.BROWSERCONFIG:
        return $localize`:@@configQuerySearchBrowserConfigsLabel:Search browser configs`;
      case Kind.BROWSERSCRIPT:
        return $localize`:@@configQuerySearchBrowserScriptsLabel:Search browser scripts`;
      case Kind.POLITENESSCONFIG:
        return $localize`:@@configQuerySearchPolitenessConfigsLabel:Search politeness configs`;
      case Kind.CRAWLHOSTGROUPCONFIG:
        return $localize`:@@configQuerySearchCrawlHostGroupsLabel:Search crawl host groups`;
      case Kind.ROLEMAPPING:
        return $localize`:@@configQuerySearchUsersLabel:Search users`;
      default:
        return $localize`:@@configQuerySearchInputLabel:Search`;
    }
  }

  override onQuery(query: ConfigQuery) {
    super.onQuery({
      term: this.term,
      ...query,
      disabled: query.disabled ?? null,
      browserScriptType: query.browserScriptType ?? null,
      robotsPolicy: query.robotsPolicy ?? null,
      role: query.role ?? null,
    });
  }

  onSearch(term: string) {
    this.term = term;
    const serializedTerm = serializeConfigSearchTerm(term, this.appliedLabelSearch);
    this.onQuery({...this.form.value, term: serializedTerm});
  }

  clearSearch(): void {
    this.term = '';
    this.onSearch('');
    this.searchElement?.nativeElement.focus();
  }

  insertLabelSearch(): void {
    this.insertSearchTerm('label:');
  }

  async chooseLabelEmoji(): Promise<void> {
    const {EmojiPickerDialogComponent} = await import('../../../../shared/components/emoji-picker/emoji-picker-dialog.component');
    this.dialog.open<InstanceType<typeof EmojiPickerDialogComponent>, void, string>(EmojiPickerDialogComponent, {
      width: '552px',
      maxWidth: 'calc(100vw - 24px)',
      autoFocus: false,
      restoreFocus: true,
    }).afterClosed().pipe(
      filter((unicode): unicode is string => !!unicode),
      take(1),
    ).subscribe(unicode => {
      const term = `label:emoji:${unicode}`;
      this.appliedLabelSearch = null;
      this.insertSearchTerm(term);
      this.onSearch(term);
    });
  }

  crawlJobRemoveLabel(name: string): string {
    return $localize`:@@configQueryRemoveCrawlJobFilterLabel:Remove ${name} crawl job filter`;
  }

  removeCrawlJob(id: string): void {
    const control = this.form.controls['crawlJobIdList'];
    const selectedIds = control.value as string[] | null;
    control.setValue((selectedIds ?? []).filter(selectedId => selectedId !== id));
  }

  protected override createForm(): void {
    this.form = this.fb.group({
      entityId: '',
      scheduleId: '',
      crawlConfigId: '',
      collectionId: '',
      browserConfigId: '',
      politenessId: '',
      crawlJobIdList: {value: [], disabled: false},
      scriptIdList: {value: [], disabled: false},
      disabled: {value: null, disabled: false},
      browserScriptType: {value: null, disabled: false},
      robotsPolicy: {value: null, disabled: false},
      role: {value: null, disabled: false},
    });
  }

  protected override updateForm(): void {
    const appliedSearch = this.query.kind === Kind.ROLEMAPPING
      ? {name: this.query.term ?? '', label: null}
      : parseConfigSearchTerm(this.query.term ?? '');
    this.appliedLabelSearch = appliedSearch.label;
    this.term = appliedSearch.label ? appliedSearch.name : this.query.term;
    super.updateForm();
  }

  private insertSearchTerm(term: string): void {
    this.term = term;
    this.cdr.markForCheck();
    this.searchElement?.nativeElement.focus();
  }

}
