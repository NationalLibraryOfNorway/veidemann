import { ChangeDetectionStrategy, Component, EventEmitter, Input, OnChanges, Output, SimpleChanges, inject } from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';
import {CustomValidators} from '../../../../../shared/validation';
import {ConfigObject, Kind, Meta, Role, RoleMapping} from '../../../../../shared/models';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {MatButtonModule} from '@angular/material/button';
import {MatInput} from '@angular/material/input';


@Component({
  selector: 'app-rolemapping-details',
  templateUrl: './rolemapping-details.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    MatButtonModule,
    MatFormFieldModule,
    MatInput,
    MatSelectModule,
    ReactiveFormsModule
  ],
  standalone: true
})

export class RoleMappingDetailsComponent implements OnChanges {
  protected fb = inject(UntypedFormBuilder);

  readonly Role = Role;

  @Input()
  configObject: ConfigObject;

  @Input()
  roles: Role[] = [];

  @Output()
  save = new EventEmitter<ConfigObject>();

  @Output()
  update = new EventEmitter<ConfigObject>();

  form: UntypedFormGroup;

  constructor() {
    this.createForm();
  }

  get showSave(): boolean {
    return this.configObject && !this.configObject.id;
  }

  get canSave(): boolean {
    return this.form.valid;
  }

  get canUpdate() {
    return (this.form.valid && this.form.dirty);
  }

  get canRevert() {
    return this.form.dirty;
  }

  get email() {
    return this.form.get('email');
  }

  get identityType() {
    return this.form.get('identityType');
  }

  get group() {
    return this.form.get('group');
  }

  get roleList() {
    return this.form.get('roleList');
  }

  ngOnChanges(changes: SimpleChanges) {
    if (changes['configObject']) {
      if (!this.configObject) {
        this.form.reset();
      } else {
        this.updateForm();
      }
    }
  }

  onSave() {
    this.save.emit(this.prepareSave());
  }

  onUpdate(): void {
    this.update.emit(this.prepareSave());
  }

  onRevert() {
    this.updateForm();
  }

  protected createForm() {
    this.form = this.fb.group({
      id: {value: '', disabled: true},
      identityType: ['email', Validators.required],
      email: ['', [Validators.required, Validators.email]],
      group: [{value: '', disabled: true}],
      roleList: [[], [Validators.required, CustomValidators.nonEmpty]]
    });
    this.identityType.valueChanges.subscribe(type => this.applyIdentityType(type));
  }

  protected updateForm() {
    this.form.patchValue({
      id: this.configObject.id,
      identityType: this.getIdentityType(),
      email: this.configObject.roleMapping.email,
      group: this.configObject.roleMapping.group,
      roleList: this.configObject.roleMapping.roleList,
    });
    this.applyIdentityType(this.identityType.value);
    this.form.markAsPristine();
    this.form.markAsUntouched();
  }

  protected prepareSave(): ConfigObject {
    const formModel = this.form.getRawValue();

    const configObject = new ConfigObject({kind: Kind.ROLEMAPPING});
    if (this.configObject.id !== '') {
      configObject.id = this.configObject.id;
    }

    const roleMapping = new RoleMapping();
    roleMapping.roleList = formModel.roleList;
    if (formModel.identityType === 'email') {
      roleMapping.email = formModel.email;
      roleMapping.group = '';
    } else {
      roleMapping.group = formModel.group;
      roleMapping.email = '';
    }

    configObject.meta = new Meta({name: 'roleMapping'});
    configObject.roleMapping = roleMapping;
    return configObject;
  }

  private getIdentityType(): 'email' | 'group' {
    return this.configObject?.roleMapping?.group ? 'group' : 'email';
  }

  private applyIdentityType(type: 'email' | 'group'): void {
    if (type === 'group') {
      this.email.reset('', {emitEvent: false});
      this.email.disable({emitEvent: false});
      this.group.enable({emitEvent: false});
      this.group.setValidators(Validators.required);
    } else {
      this.group.reset('', {emitEvent: false});
      this.group.disable({emitEvent: false});
      this.email.enable({emitEvent: false});
      this.email.setValidators([Validators.required, Validators.email]);
    }
    this.email.updateValueAndValidity({emitEvent: false});
    this.group.updateValueAndValidity({emitEvent: false});
  }
}
