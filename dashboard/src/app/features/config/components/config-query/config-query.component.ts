import {
  AfterViewInit,
  ChangeDetectionStrategy,
  Component,
  ElementRef,
  Input,
  OnChanges,
  ViewChild
} from '@angular/core';
import {UntypedFormBuilder} from '@angular/forms';

import {Kind} from '../../../shared/models';
import {ConfigQuery} from '../../../shared/func';
import {ConfigOptions} from '../../func';
import {QueryComponent} from '../../../commons/components';
import {ShortcutEventOutput, ShortcutInput} from 'ng-keyboard-shortcuts';


@Component({
    selector: 'app-config-query',
    styleUrls: ['config-query.component.scss'],
    templateUrl: './config-query.component.html',
    changeDetection: ChangeDetectionStrategy.OnPush,
    standalone: true
})
export class ConfigQueryComponent extends QueryComponent<ConfigQuery> implements OnChanges, AfterViewInit {
  readonly Kind = Kind;
  shortcuts: ShortcutInput[] = [];

  term: string;

  @Input()
  options: ConfigOptions;

  @ViewChild('search') searchElement: ElementRef;

  constructor(protected override fb: UntypedFormBuilder) {
    super(fb);
  }

  override ngAfterViewInit() {
    super.ngAfterViewInit();
    this.shortcuts.push(
      {
        key: 'shift + l',
        label: 'Query',
        description: 'Focus query search input',
        command: (output: ShortcutEventOutput) => {
          event.preventDefault();
          this.searchElement.nativeElement.focus();
        }
      },
    );
  }

  override onQuery(query: ConfigQuery) {
    super.onQuery({term: this.term, ...query});
  }

  onSearch(term: string) {
    this.onQuery({...this.form.value, term});
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
    });
  }

  protected override updateForm(): void {
    this.term = this.query.term;
    super.updateForm();
  }
}
