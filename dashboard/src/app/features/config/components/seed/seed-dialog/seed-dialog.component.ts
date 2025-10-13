import {ChangeDetectionStrategy, Component, Inject, OnInit} from '@angular/core';
import {ReactiveFormsModule, UntypedFormBuilder} from '@angular/forms';
import {AuthService} from '../../../../../core/auth';
import {ConfigObject} from '../../../../../shared/models';
import {SeedDetailsComponent} from '..';
import {MAT_DIALOG_DATA, MatDialogModule, MatDialogRef} from '@angular/material/dialog';
import {ConfigDialogData} from '../../../func';
import {MatSlideToggle} from '@angular/material/slide-toggle';
import {SeedMetaComponent} from '../../seed-meta/seed-meta.component';
import {MatFormFieldModule} from '@angular/material/form-field';
import {MatSelectModule} from '@angular/material/select';
import {MatButtonModule} from '@angular/material/button';
import {LayoutGapDirective} from '@ngbracket/ngx-layout/flex';
import {FlexDirective} from '@ngbracket/ngx-layout';

@Component({
  selector: 'app-entity-dialog',
  templateUrl: './seed-dialog.component.html',
  changeDetection: ChangeDetectionStrategy.OnPush,
  imports: [
    FlexDirective,
    LayoutGapDirective,
    MatButtonModule,
    MatDialogModule,
    MatFormFieldModule,
    MatSelectModule,
    MatSlideToggle,
    ReactiveFormsModule,
    SeedMetaComponent
  ],
  standalone: true
})
export class SeedDialogComponent extends SeedDetailsComponent implements OnInit {
  declare crawlJobs: ConfigObject[];

  constructor(protected override fb: UntypedFormBuilder,
              protected override authService: AuthService,
              @Inject(MAT_DIALOG_DATA) private data: ConfigDialogData,
              public dialogRef: MatDialogRef<SeedDialogComponent>) {
    super(fb, authService);
    this.createForm();
    this.crawlJobs = data.options.crawlJobs;
    this.configObject = data.configObject;
  }

  ngOnInit(): void {
    this.updateForm();
  }

  onDialogClose(): ConfigObject | ConfigObject[] {
    return this.isMultipleSeed()
      ? this.prepareSaveMultiple()
      : this.prepareSave();
  }
}
