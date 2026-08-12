import {ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {ConfigObject, Kind, Meta} from '../../../../../shared/models';
import {MetaComponent} from '../../meta/meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatInputModule} from '@angular/material/input';
import {MatButtonModule} from '@angular/material/button';
import {MatIcon} from '@angular/material/icon';
import {MatTooltip} from '@angular/material/tooltip';
import {CopyIdDirective} from '../../../../../shared/directives';

@Component({
  selector: 'app-entity-details',
  templateUrl: './entity-details.component.html',
  styleUrls: ['./entity-details.component.scss'],
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    CopyIdDirective,
    MatButtonModule,
    MatFormFieldModule,
    MatInputModule,
    MatIcon,
    MatTooltip,
    MetaComponent,
    ReactiveFormsModule,
  ],
  standalone: true
})

export class EntityDetailsComponent implements OnChanges {
  protected fb = inject(UntypedFormBuilder);
  protected authService = inject(AuthService);


  @Input()
  configObject: ConfigObject;

  @Input()
  annotationSuggestions: string[] = [];

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  update = new EventEmitter<ConfigObject>();


  form: UntypedFormGroup;

  constructor() {
    this.createForm();
  }

  get showSave(): boolean {
    return this.configObject ? !this.configObject.id : false;
  }

  get canSave(): boolean {
    return this.form.valid;
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

  ngOnChanges(changes: SimpleChanges): void {
    if (changes['configObject']) {
      if (this.configObject) {
        this.updateForm();
      } else {
        this.form.reset();
      }
    }
  }

  onSave() {
    this.save.emit(this.prepareSave());
  }

  onUpdate() {
    this.update.emit(this.prepareSave());
  }

  onRevert() {
    this.updateForm();
  }

  protected createForm() {
    this.form = this.fb.group({
      id: '',
      meta: new Meta(),
    });
  }

  protected updateForm() {
    this.form.patchValue({
      id: this.configObject.id,
      meta: this.configObject.meta
    });
    this.form.markAsPristine();
    this.form.markAsUntouched();

    if (!this.canEdit) {
      this.form.disable();
    }
  }

  protected prepareSave(): ConfigObject {
    const formModel = this.form.value;

    return new ConfigObject({
      id: formModel.id,
      meta: formModel.meta,
      kind: Kind.CRAWLENTITY
    });
  }
}
