import { AfterViewInit,ChangeDetectionStrategy,Component,ElementRef,Input,OnChanges,ViewChild } from '@angular/core';
import { FormsModule,ReactiveFormsModule } from '@angular/forms';

import { MatButtonModule } from '@angular/material/button';
import { MatChipsModule } from '@angular/material/chips';
import { MatFormFieldModule } from '@angular/material/form-field';
import { MatIcon } from '@angular/material/icon';
import { MatInput } from '@angular/material/input';
import { MatSelectModule } from '@angular/material/select';
import { QueryComponent } from '../../../../shared/components';
import { ConfigQuery } from '../../../../shared/func';
import { BrowserScriptType, Kind } from '../../../../shared/models';
import { ConfigOptions } from '../../func';

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
    ReactiveFormsModule
  ],
  standalone: true
})
export class ConfigQueryComponent extends QueryComponent<ConfigQuery> implements OnChanges, AfterViewInit {

  readonly Kind = Kind;
  readonly BrowserScriptType = BrowserScriptType;
  readonly labelSearchSuggestions = [
    {
      label: $localize`:@@configQueryExactLabelSuggestion:Exact label`,
      term: 'label:type:default',
    },
    {
      label: $localize`:@@configQueryLabelKeySuggestion:Label key`,
      term: 'label:type:',
    },
    {
      label: $localize`:@@configQueryLabelValuePrefixSuggestion:Value prefix`,
      term: 'label:type:def*',
    },
  ];

  term: string;

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

  get showLabelSearchSuggestions(): boolean {
    return !!this.query?.kind && this.query.kind !== Kind.ROLEMAPPING;
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
    });
  }

  onSearch(term: string) {
    this.term = term;
    this.onQuery({...this.form.value, term});
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
    });
  }

  protected override updateForm(): void {
    this.term = this.query.term;
    super.updateForm();
  }
}
