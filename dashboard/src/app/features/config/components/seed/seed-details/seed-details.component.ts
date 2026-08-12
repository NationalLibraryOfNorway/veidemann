import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, OnDestroy, Output, SimpleChanges, inject } from '@angular/core';
import {AbstractControl, ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';


import {AuthService} from '../../../../../core/auth';
import {ConfigObject, ConfigRef, Kind, Meta} from '../../../../../shared/models';
import {Subject} from 'rxjs';
import {EffectiveScriptAnnotationsComponent, Parcel, SeedMetaComponent} from '../..';
import {configRefIdRequired} from '../../../../../shared/validation/configref';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatButtonModule} from '@angular/material/button';
import {MatSelectModule} from '@angular/material/select';
import {MatInput} from '@angular/material/input';
import {CopyIdDirective} from '../../../../../shared/directives';
import {MatDialog} from '@angular/material/dialog';
import {filter, takeUntil} from 'rxjs/operators';
import {
  MoveSeedDialogComponent,
  MoveSeedDialogData,
  MoveSeedDialogResult,
} from '../move-seed-dialog/move-seed-dialog.component';
import {MatCheckboxModule} from '@angular/material/checkbox';

@Component({
  selector: 'app-seed-details',
  templateUrl: './seed-details.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CopyIdDirective,
    MatButtonModule,
    MatCheckboxModule,
    MatFormFieldModule,
    MatIcon,
    MatInput,
    MatSelectModule,
    MatTooltip,
    ReactiveFormsModule,
    EffectiveScriptAnnotationsComponent,
    SeedMetaComponent,
  ],
  standalone: true
})
export class SeedDetailsComponent implements OnChanges, OnDestroy {
  protected fb = inject(UntypedFormBuilder);
  protected authService = inject(AuthService);
  private dialog = inject(MatDialog);


  @Input()
  configObject: ConfigObject;

  @Input()
  crawlJobs: ConfigObject[];

  @Input()
  annotationSuggestions: string[] = [];

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  saveMultiple = new EventEmitter<ConfigObject[]>();

  @Output()
  update = new EventEmitter<ConfigObject>();

  @Output()
  move = new EventEmitter<Parcel>();

  // noinspection ReservedWordAsName
  @Output()
  runCrawl = new EventEmitter<ConfigObject>();

  form: UntypedFormGroup;

  ngUnsubscribe = new Subject<void>();

  constructor() {
    this.createForm();
  }

  get showSave(): boolean {
    return (this.configObject && !this.configObject.id);
  }

  get canSave(): boolean {
    return !(
      this.entityRef.get('id').hasError('required')
      || this.meta.hasError('required')
      || this.meta.hasError('pattern')
      || this.meta.pending
    );
  }

  get canUpdate(): boolean {
    return this.form.valid && this.form.dirty;
  }

  get canRevert(): boolean {
    return this.form.dirty;
  }

  get canEdit(): boolean {
    return this.authService.canUpdate(this.configObject.kind);
  }

  get entityRef(): AbstractControl {
    return this.form.get('entityRef');
  }

  get jobRefListId(): AbstractControl {
    return this.form.get('jobRefListId');
  }

  get meta(): AbstractControl {
    return this.form.get('meta');
  }

  get disabled(): AbstractControl {
    return this.form.get('disabled');
  }

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['configObject']) {
      if (this.configObject) {
        this.updateForm();
      } else {
        this.form.reset();
      }
    }
  }

  ngOnDestroy(): void {
    this.ngUnsubscribe.next();
    this.ngUnsubscribe.complete();
  }

  getCrawlJobName(id) {
    const found = this.crawlJobs.find(crawlJob => crawlJob.id === id);
    return found ? found.meta.name : 'crawlJob';
  }

  onSave(): void {
    if (this.isMultipleSeed()) {
      this.saveMultiple.emit(this.prepareSaveMultiple());
    } else {
      this.save.emit(this.prepareSave());
    }
  }

  onUpdate(): void {
    this.update.emit(this.prepareSave());
  }

  onRevert() {
    this.updateForm();
  }

  onMoveSeed(): void {
    if (!this.configObject?.id || this.form.dirty) {
      return;
    }

    this.dialog.open<MoveSeedDialogComponent, MoveSeedDialogData, MoveSeedDialogResult>(MoveSeedDialogComponent, {
      data: {seed: this.configObject},
      autoFocus: true,
    }).afterClosed().pipe(
      filter((entityRef): entityRef is ConfigRef => !!entityRef),
      takeUntil(this.ngUnsubscribe),
    ).subscribe(entityRef => this.move.emit({seed: this.configObject, entityRef}));
  }

  get moveSeedTooltip(): string {
    return this.form.dirty
      ? $localize`:@@seedDetailsMoveRequiresCleanFormTooltip:Save or revert changes before moving this seed`
      : $localize`:@@seedDetailsEditEntityIdButtonTooltip:Move seed to another entity`;
  }

  onRunCrawl(): void {
    this.runCrawl.emit(this.configObject);
  }


  protected createForm() {
    this.form = this.fb.group({
      id: '',
      disabled: '',
      entityRef: this.fb.group({
        kind: '',
        id: '',
      }, {validator: configRefIdRequired}),
      jobRefListId: {value: [], disabled: false},
      meta: new Meta(),
    });
  }

  protected updateForm() {
    if (!this.canEdit) {
      this.form.disable();
    }

    this.form.setValue({
      id: this.configObject.id,
      disabled: !!this.configObject.seed.disabled,
      entityRef: this.configObject.seed.entityRef,
      jobRefListId: this.configObject.seed.jobRefList.map(job => job.id),
      meta: this.configObject.meta,
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();
  }

  /**
   * Disabled values in form must be copied from model and not the view model (form.value)
   */
  protected prepareSave(): ConfigObject {
    const formModel = this.form.value;
    return new ConfigObject({
      id: formModel.id,
      kind: Kind.SEED,
      meta: formModel.meta,
      seed: {
        disabled: formModel.disabled,
        entityRef: formModel.entityRef,
        jobRefList: formModel.jobRefListId.map(id => new ConfigRef({kind: Kind.CRAWLJOB, id})),
      }
    });
  }

  protected prepareSaveMultiple(): ConfigObject[] {
    const formModel = this.form.value;

    const configObjectTemplate = new ConfigObject({
      id: formModel.id,
      kind: Kind.SEED,
      meta: formModel.meta,
      seed: {
        disabled: formModel.disabled,
        entityRef: formModel.entityRef,
        jobRefList: formModel.jobRefListId.map(id => new ConfigRef({kind: Kind.CRAWLJOB, id})),
      }
    });

    const urls = formModel.meta.name.trim().split(/\s+/).filter(s => !!s);

    return urls.map(url => {
      const seed = ConfigObject.clone(configObjectTemplate);
      seed.meta.name = url;
      return seed;
    });
  }

  protected isMultipleSeed() {
    const meta = this.meta.value;
    const parts = meta.name.trim().split(/\s+/);
    return parts.length > 1;
  }
}
