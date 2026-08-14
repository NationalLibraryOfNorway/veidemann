import { AfterViewInit,ChangeDetectionStrategy,ChangeDetectorRef,Component,ElementRef,inject,Input,OnChanges,SimpleChanges,ViewChild } from '@angular/core';
import {ReactiveFormsModule} from '@angular/forms';

import {MatAutocompleteModule, MatAutocompleteSelectedEvent, MatAutocompleteTrigger} from '@angular/material/autocomplete';
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
import {LabelService} from '../../services/label.service';

@Component({
  selector: 'app-config-query',
  styleUrls: ['config-query.component.scss'],
  templateUrl: './config-query.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatAutocompleteModule,
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
  private readonly labelService = inject(LabelService);

  readonly Kind = Kind;
  readonly BrowserScriptType = BrowserScriptType;
  readonly RobotsPolicy = RobotsPolicy;
  readonly Role = Role;

  term: string;
  appliedLabelSearch: ConfigLabelSelector | null = null;
  labelKeys: string[] = [];
  private labelKeysKind: Kind | null = null;
  private loadingLabelKeysKind: Kind | null = null;

  @Input()
  options: ConfigOptions;

  @Input()
  disabled = false;

  get filterBrowserScriptTypes(): BrowserScriptType[] {
    return (this.options?.browserScriptTypes ?? [])
      .filter(type => type !== BrowserScriptType.UNDEFINED);
  }

  @ViewChild('search') searchElement: ElementRef;
  @ViewChild(MatAutocompleteTrigger) labelAutocompleteTrigger: MatAutocompleteTrigger;

  get filteredLabelKeys(): string[] {
    const fragment = this.labelKeyFragment(this.term);
    if (fragment === null) {
      return [];
    }
    const normalizedFragment = fragment.toLowerCase();
    return this.labelKeys.filter(key => key.toLowerCase().startsWith(normalizedFragment));
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

  override ngOnChanges(changes: SimpleChanges): void {
    super.ngOnChanges(changes);
    if (changes['disabled']) {
      if (this.disabled) {
        this.form.disable({emitEvent: false});
        this.labelAutocompleteTrigger?.closePanel();
      } else {
        this.form.enable({emitEvent: false});
      }
    }
  }

  override onQuery(query: ConfigQuery) {
    const term = Object.prototype.hasOwnProperty.call(query, 'term')
      ? query.term
      : serializeConfigSearchTerm(this.term, this.appliedLabelSearch);
    super.onQuery({
      ...this.query,
      ...query,
      term,
      disabled: query.disabled ?? null,
      browserScriptType: query.browserScriptType ?? null,
      robotsPolicy: query.robotsPolicy ?? null,
      role: query.role ?? null,
    });
  }

  onSearch(term: string) {
    if (this.disabled) {
      return;
    }
    this.term = term;
    const serializedTerm = serializeConfigSearchTerm(term, this.appliedLabelSearch);
    this.onQuery({...this.form.value, term: serializedTerm});
  }

  clearSearch(): void {
    if (this.disabled) {
      return;
    }
    this.term = '';
    this.onSearch('');
    this.searchElement?.nativeElement.focus();
  }

  insertLabelSearch(): void {
    if (this.disabled) {
      return;
    }
    this.insertSearchTerm('label:');
    this.activateLabelAutocomplete();
  }

  onSearchTermChange(term: string): void {
    if (this.disabled) {
      return;
    }
    this.term = term;
    this.activateLabelAutocomplete();
  }

  onLabelKeySelected(event: MatAutocompleteSelectedEvent): void {
    if (this.disabled) {
      return;
    }
    const marker = 'label:';
    const markerIndex = this.term.indexOf(marker);
    if (markerIndex < 0) {
      return;
    }
    this.term = `${this.term.slice(0, markerIndex + marker.length)}${event.option.value}:`;
    this.cdr.markForCheck();
    this.searchElement?.nativeElement.focus();
  }

  async chooseLabelEmoji(): Promise<void> {
    if (this.disabled) {
      return;
    }
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

  private activateLabelAutocomplete(): void {
    if (this.disabled) {
      this.labelAutocompleteTrigger?.closePanel();
      return;
    }
    const kind = this.query?.kind;
    if (!kind || kind === Kind.ROLEMAPPING || this.labelKeyFragment(this.term) === null) {
      this.labelAutocompleteTrigger?.closePanel();
      return;
    }

    if (this.labelKeysKind === kind) {
      this.openLabelAutocomplete();
      return;
    }
    if (this.loadingLabelKeysKind === kind) {
      return;
    }

    this.loadingLabelKeysKind = kind;
    this.labelService.getLabelKeys(kind).pipe(take(1)).subscribe(keys => {
      if (this.loadingLabelKeysKind === kind) {
        this.loadingLabelKeysKind = null;
      }
      if (this.query?.kind !== kind) {
        return;
      }
      this.labelKeysKind = kind;
      this.labelKeys = keys;
      this.openLabelAutocomplete();
    });
  }

  private openLabelAutocomplete(): void {
    this.cdr.markForCheck();
    Promise.resolve().then(() => {
      if (this.filteredLabelKeys.length && this.searchElement?.nativeElement.matches(':focus')) {
        this.labelAutocompleteTrigger?.openPanel();
      }
    });
  }

  private labelKeyFragment(term: string): string | null {
    const marker = 'label:';
    const markerIndex = term?.indexOf(marker) ?? -1;
    if (markerIndex < 0) {
      return null;
    }
    const fragment = term.slice(markerIndex + marker.length);
    return fragment.includes(':') ? null : fragment;
  }

}
